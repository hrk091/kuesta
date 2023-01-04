/*
 Copyright (c) 2023 NTT Communications Corporation

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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"runtime/debug"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show kuesta version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd.Context())
		},
	}
	return cmd
}

func runVersion(ctx context.Context) error {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.WithStack(fmt.Errorf("get buildinfo"))
	}
	b, err := json.Marshal(map[string]string{
		"version":   version,
		"gitCommit": commit,
		"date":      date,
		"goVersion": buildInfo.GoVersion,
	})
	if err != nil {
		return errors.WithStack(fmt.Errorf("marshal version info: %w", err))
	}
	fmt.Printf("%s\n", b)
	return nil
}