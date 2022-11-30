/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package kuesta

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/nttcom/kuesta/pkg/common"
	"github.com/nttcom/kuesta/pkg/gogit"
	"github.com/nttcom/kuesta/pkg/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	UpdateCheckDuration = 5 * time.Second
)

type DeviceAggregateCfg struct {
	RootCfg

	Addr string
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

	l.Infof("Start simple api server on %s", cfg.Addr)
	http.HandleFunc("/commit", s.HandleFunc)
	if err := http.ListenAndServe(cfg.Addr, nil); err != nil {
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
	if st := logger.GetStackTrace(err); st != "" {
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
					logger.Error(ctx, err, "save actual device config")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	l.Info("Start saver loop")
}

func (s *DeviceAggregateServer) runCommitter(ctx context.Context) {
	common.SetInterval(ctx, func() {
		if err := s.GitPushDeviceConfig(ctx); err != nil {
			logger.Error(ctx, err, "push sync branch")
		}
	}, UpdateCheckDuration)
}

// SaveConfig writes device config contained in supplied SaveConfigRequest.
func (s *DeviceAggregateServer) SaveConfig(ctx context.Context, r *SaveConfigRequest) error {
	dp := DevicePath{RootDir: s.cfg.StatusRootPath, Device: r.Device}
	if err := WriteFileWithMkdir(dp.DeviceActualConfigPath(IncludeRoot), []byte(*r.Config)); err != nil {
		return fmt.Errorf("write actual device config: %w", err)
	}
	return nil
}

// GitPushDeviceConfig runs git-commit all unstaged device config updates as batch commit then git-push to remote origin.
func (s *DeviceAggregateServer) GitPushDeviceConfig(ctx context.Context) error {
	l := logger.FromContext(ctx)

	g, err := gogit.NewGit(s.cfg.StatusGitOptions().ShouldCloneIfNotExist())
	if err != nil {
		return fmt.Errorf("init git: %w", err)
	}

	if err := g.Pull(); err != nil {
		return fmt.Errorf("git pull: %w", err)
	}

	w, err := g.Checkout()
	if err != nil {
		return fmt.Errorf("git checkout to trunk: %w", err)
	}

	if err := g.RemoveGoneBranches(); err != nil {
		return fmt.Errorf("remove gone branches: %w", err)
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

	commitMsg := MakeSyncCommitMessage(stmap)
	if _, err := g.Commit(commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	l.Infof("commited: %s\n", commitMsg)
	if err := g.Push(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
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