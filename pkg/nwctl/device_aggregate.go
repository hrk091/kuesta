/*
 Copyright 2022 NTT Communications Corporation.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package nwctl

import (
	"context"
	"cuelang.org/go/pkg/strconv"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/gogit"
	"github.com/hrk091/nwctl/pkg/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	UpdateCheckDuration = 5 * time.Second
)

type DeviceAggregateCfg struct {
	RootCfg

	Port string
}

// Validate validates exposed fields according to the `validate` tag.
func (c *DeviceAggregateCfg) Validate() error {
	return common.Validate(c)
}

// RunDeviceAggregate runs the main process of the `device aggregate` command.
func RunDeviceAggregate(ctx context.Context, cfg *DeviceAggregateCfg) error {
	l := logger.FromContext(ctx)
	l.Debug("device aggregate called")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s := NewDeviceAggregateServer(cfg)
	s.Run(ctx)

	l.Infof("Start simple api server on %s", cfg.Port)
	http.HandleFunc("/commit", s.HandleFunc)
	if err := http.ListenAndServe(cfg.Port, nil); err != nil {
		return err
	}
	return nil
}

// DeviceAggregateServer runs saver loop and committer loop along with serving commit API to persist device config to git.
// Device config are written locally and added to git just after commit API call. Updated configs are aggregated
// and git-pushed as batch commit periodically.
type DeviceAggregateServer struct {
	ch  chan *SaveConfigRequest
	cfg *DeviceAggregateCfg
}

// NewDeviceAggregateServer creates new DeviceAggregateServer.
func NewDeviceAggregateServer(cfg *DeviceAggregateCfg) *DeviceAggregateServer {
	return &DeviceAggregateServer{
		ch:  make(chan *SaveConfigRequest),
		cfg: cfg,
	}
}

func (s *DeviceAggregateServer) Error(l *zap.SugaredLogger, err error, msg string, kvs ...interface{}) {
	l = l.WithOptions(zap.AddCallerSkip(1))
	if st := common.GetStackTrace(err); st != "" {
		l = l.With("stacktrace", st)
	}
	l.Errorw(fmt.Sprintf("%s: %v", msg, err), kvs...)
}

// HandleFunc handles API call to persist actual device config.
func (s *DeviceAggregateServer) HandleFunc(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		ctx := r.Context()
		if err, code := s.add(ctx, r.Body); err != nil {
			http.Error(w, err.Error(), code)
		}
		defer r.Body.Close()
		return
	default:
		http.Error(w, `{"status": "only POST allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *DeviceAggregateServer) add(ctx context.Context, r io.Reader) (error, int) {
	req, err := DecodeSaveConfigRequest(r)
	if err != nil {
		return err, 400
	}
	s.ch <- req
	return nil, 200
}

func (s *DeviceAggregateServer) Run(ctx context.Context) {
	s.runSaver(ctx)
	s.runCommitter(ctx)
}

func (s *DeviceAggregateServer) runSaver(ctx context.Context) {
	l := logger.FromContext(ctx)

	go func() {
		for {
			select {
			case r := <-s.ch:
				l.Infof("update received: device=%s", r.Device)
				if err := s.SaveConfig(ctx, r); err != nil {
					s.Error(l, err, "save actual device config")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	l.Info("Start saver loop")
}

func (s *DeviceAggregateServer) runCommitter(ctx context.Context) {
	l := logger.FromContext(ctx)

	go func() {
		for {
			select {
			case <-time.After(UpdateCheckDuration):
				l.Info("Checking git status...")
				if err := s.GitPushSyncBranch(ctx); err != nil {
					s.Error(l, err, "push sync branch")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	l.Info("Start committer loop")
}

// SaveConfig writes device config contained in supplied SaveConfigRequest.
func (s *DeviceAggregateServer) SaveConfig(ctx context.Context, r *SaveConfigRequest) error {
	dp := DevicePath{RootDir: s.cfg.RootPath, Device: r.Device}
	if err := WriteFileWithMkdir(dp.DeviceActualConfigPath(IncludeRoot), []byte(*r.Config)); err != nil {
		return fmt.Errorf("write actual device config: %w", err)
	}
	return nil
}

// GitPushSyncBranch runs git-commit all unstaged device config updates as batch commit then git-push to remote origin.
func (s *DeviceAggregateServer) GitPushSyncBranch(ctx context.Context) error {
	l := logger.FromContext(ctx)

	g, err := gogit.NewGit(s.cfg.GitOptions().ShouldCloneIfNotExist())
	if err != nil {
		return fmt.Errorf("init git: %w", err)
	}

	if _, err := g.Checkout(); err != nil {
		return fmt.Errorf("git checkout to trunk: %w", err)
	}

	if err := g.Pull(); err != nil {
		return fmt.Errorf("git pull: %w", err)
	}

	if err := g.RemoveGoneBranches(); err != nil {
		return fmt.Errorf("remove gone branches: %w", err)
	}

	branches, err := g.Branches()
	if err != nil {
		return fmt.Errorf("get local branches: %w", err)
	}
	latestSyncBr := LatestSyncBranch(branches)

	var opt gogit.CheckoutOpts
	if latestSyncBr != "" {
		opt = gogit.CheckoutOptsTo(latestSyncBr)
	}
	w, err := g.Checkout(opt)
	if err != nil {
		return fmt.Errorf("git checkout to %s: %w", latestSyncBr, err)
	}

	_, err = w.Add("devices")
	if err != nil {
		return fmt.Errorf("git add devices: %v", err)
	}

	stmap, err := w.Status()
	if err != nil {
		return fmt.Errorf("get status map: %v", err)
	}
	// TODO check only staged files
	if len(stmap) == 0 {
		l.Info("skipped: there are no update")
		return nil
	}
	if err := CheckGitIsStagedOrUnmodified(stmap); err != nil {
		return fmt.Errorf("check files are either staged or unmodified: %w", err)
	}

	var toOpts []gogit.CheckoutOpts
	toBranch := latestSyncBr
	if toBranch == "" {
		toBranch = fmt.Sprintf("SYNC-%d", time.Now().Unix())
		toOpts = append(toOpts, gogit.CheckoutOptsCreateNew(), gogit.CheckoutOptsTo(toBranch))
	} else {
		toOpts = append(toOpts, gogit.CheckoutOptsTo(latestSyncBr))
	}
	if w, err = g.Checkout(toOpts...); err != nil {
		return fmt.Errorf("create new branch: %w", err)
	}

	commitMsg := MakeSyncCommitMessage(stmap)
	if _, err := g.Commit(commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	l.Infof("commited: %s\n", commitMsg)
	if err := g.Push(gogit.PushOptBranch(toBranch)); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

// LatestSyncBranch returns the latest sync branch name or empty string if there are no sync branch.
func LatestSyncBranch(branches []*plumbing.Reference) string {
	re := regexp.MustCompile(`SYNC-(\d+)`)

	var max int
	for _, b := range branches {
		matched := re.FindStringSubmatch(b.Name().Short())
		if len(matched) == 0 {
			continue
		}
		ts, _ := strconv.Atoi(matched[1])
		if ts > max {
			max = ts
		}
	}
	if max != 0 {
		return fmt.Sprintf("SYNC-%d", max)
	}
	return ""
}

type SaveConfigRequest struct {
	Device string  `json:"device" validate:"required"`
	Config *string `json:"config" validate:"required"`
}

func (r *SaveConfigRequest) Validate() error {
	return common.Validate(r)
}

// DecodeSaveConfigRequest decodes supplied payload to SaveConfigRequest.
func DecodeSaveConfigRequest(r io.Reader) (*SaveConfigRequest, error) {
	var req SaveConfigRequest
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return nil, fmt.Errorf("decode: %v", err)
	}
	return &req, req.Validate()
}

// MakeSyncCommitMessage returns the commit message that shows the device actual config updates.
func MakeSyncCommitMessage(stmap git.Status) string {
	var devicesAdded []string
	var devicesModified []string
	var devicesDeleted []string

	for path, st := range stmap {
		dir, file := filepath.Split(path)
		dirElem := strings.Split(dir, string(filepath.Separator))
		if dirElem[0] == "devices" && file == "actual_config.cue" {
			deviceName := dirElem[1]
			switch st.Staging {
			case git.Added:
				devicesAdded = append(devicesAdded, deviceName)
			case git.Modified:
				devicesModified = append(devicesModified, deviceName)
			case git.Deleted:
				devicesDeleted = append(devicesDeleted, deviceName)
			}
		}
	}
	for _, v := range [][]string{devicesAdded, devicesModified, devicesDeleted} {
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	}

	devices := append(devicesAdded, devicesDeleted...)
	devices = append(devices, devicesModified...)

	title := fmt.Sprintf("Updated: %s", strings.Join(devices, " "))
	var bodylines []string
	bodylines = append(bodylines, "", "Devices:")
	for _, d := range devicesAdded {
		bodylines = append(bodylines, fmt.Sprintf("\tadded:     %s", d))
	}
	for _, d := range devicesDeleted {
		bodylines = append(bodylines, fmt.Sprintf("\tdeleted:   %s", d))
	}
	for _, d := range devicesModified {
		bodylines = append(bodylines, fmt.Sprintf("\tmodified:  %s", d))
	}

	return title + "\n" + strings.Join(bodylines, "\n")
}
