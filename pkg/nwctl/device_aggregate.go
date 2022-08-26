/*
 * Copyright (c) 2022. Hiroki Okui
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package nwctl

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/gogit"
	"github.com/hrk091/nwctl/pkg/logger"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

	s := NewDeviceAggregateServer()
	s.runSaver(ctx)
	s.runCommitter(ctx)

	l.Infof("Start simple api server on %s", cfg.Port)
	http.HandleFunc("/commit", s.HandleFunc)
	if err := http.ListenAndServe(cfg.Port, nil); err != nil {
		return err
	}
	return nil
}

type DeviceAggregateServer struct {
	ch  chan *SaveConfigRequest
	cfg DeviceAggregateCfg
}

func NewDeviceAggregateServer() *DeviceAggregateServer {
	return &DeviceAggregateServer{
		ch: make(chan *SaveConfigRequest),
	}
}

func (s *DeviceAggregateServer) HandleFunc(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	switch r.Method {
	case http.MethodPost:
		if err, code := s.add(ctx, r.Body); err != nil {
			http.Error(w, err.Error(), code)
		}
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

func (s *DeviceAggregateServer) runSaver(ctx context.Context) {
	l := logger.FromContext(ctx)

	go func() {
		for {
			select {
			case r := <-s.ch:
				l.Infof("update received: device=%s", r.Device)
				if err := s.SaveConfig(ctx, r); err != nil {
					l.Errorf("save actual device config: %v", err)
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
			case <-time.After(5 * time.Second):
				l.Info("Checking git status...")
				if err := s.GitPushSyncBranch(ctx); err != nil {
					l.Errorf("push sync branch: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	l.Info("Start committer loop")
}

func (s *DeviceAggregateServer) SaveConfig(ctx context.Context, r *SaveConfigRequest) error {
	dp := DevicePath{RootDir: s.cfg.RootPath, Device: r.Device}
	if err := WriteFileWithMkdir(dp.DeviceActualConfigPath(IncludeRoot), []byte(r.Config)); err != nil {
		return fmt.Errorf("write actual device config: %w", err)
	}
	return nil
}

func (s *DeviceAggregateServer) GitPushSyncBranch(ctx context.Context) error {
	l := logger.FromContext(ctx)

	g, err := gogit.NewGit(gogit.GitOptions{
		Path:        s.cfg.RootPath,
		TrunkBranch: s.cfg.GitTrunk,
		RemoteName:  s.cfg.GitRemote,
		Token:       s.cfg.GitToken,
		User:        s.cfg.GitUser,
		Email:       s.cfg.GitEmail,
	})
	if err != nil {
		return fmt.Errorf("setup git: %w", err)
	}

	if err := g.Pull(); err != nil {
		return fmt.Errorf("git pull: %w", err)
	}
	w, err := g.Checkout()
	if err != nil {
		return fmt.Errorf("git checkout: %w", err)
	}
	_, err = w.Add("devices")
	if err != nil {
		return fmt.Errorf("git add devices: %v", err)
	}

	stmap, err := w.Status()
	if err != nil {
		return fmt.Errorf("unable to get status map: %v", err)
	}
	if len(stmap) == 0 {
		l.Info("skipped: there are no update")
		return nil
	}
	if err := CheckGitIsStagedOrUnmodified(stmap); err != nil {
		return fmt.Errorf("check files are either staged or unmodified: %w", err)
	}

	branchName := fmt.Sprintf("SYNC-%d", time.Now().Unix())
	if w, err = g.Checkout(gogit.CheckoutOptsTo(branchName), gogit.CheckoutOptsCreateNew()); err != nil {
		return fmt.Errorf("create new branch: %w", err)
	}

	commitMsg := MakeSyncCommitMessage(stmap)
	if _, err := g.Commit(commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	if err := g.Push(branchName); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

type SaveConfigRequest struct {
	Device string `json:"device"`
	Config string `json:"config"`
}

func DecodeSaveConfigRequest(r io.Reader) (*SaveConfigRequest, error) {
	var req SaveConfigRequest
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return nil, fmt.Errorf("unable to decode request body: %v", err)
	}
	if req.Device == "" {
		return nil, fmt.Errorf("device name is not given")
	}
	if req.Config == "" {
		return nil, fmt.Errorf("device config is not given")
	}
	return &req, nil
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

	title := fmt.Sprintf("Updated: %s", strings.Join(devices, ","))
	var bodylines []string
	bodylines = append(bodylines, "", "Devices:")
	for _, d := range devicesAdded {
		bodylines = append(bodylines, fmt.Sprintf("    added:     %s", d))
	}
	for _, d := range devicesDeleted {
		bodylines = append(bodylines, fmt.Sprintf("    deleted:   %s", d))
	}
	for _, d := range devicesModified {
		bodylines = append(bodylines, fmt.Sprintf("    modified:  %s", d))
	}

	return title + "\n" + strings.Join(bodylines, "\n")
}
