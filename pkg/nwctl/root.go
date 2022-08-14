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
	"go.uber.org/multierr"
)

type RootCfg struct {
	Verbose  uint8
	Devel    bool
	RootPath string
}

// NewRootCfg creates new RootCfg with given options.
func NewRootCfg(opts ...RootCfgOpts) (*RootCfg, error) {
	cfg := &RootCfg{}
	var err error
	for _, opt := range opts {
		err = multierr.Append(err, opt(cfg))
	}
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

type RootCfgOpts func(cfg *RootCfg) error

// Verbose sets verbose parameter to RootCfg.
func Verbose(v uint8) RootCfgOpts {
	return func(cfg *RootCfg) error {
		if v > 3 {
			return &ErrConfigValue{"verbose must be less than 4"}
		}
		cfg.Verbose = v
		return nil
	}
}

// Devel sets devel parameter to RootCfg.
func Devel(v bool) RootCfgOpts {
	return func(cfg *RootCfg) error {
		cfg.Devel = v
		return nil
	}
}

// RootPath sets rootpath parameter to RootCfg.
func RootPath(v string) RootCfgOpts {
	return func(cfg *RootCfg) error {
		cfg.RootPath = v
		return nil
	}
}
