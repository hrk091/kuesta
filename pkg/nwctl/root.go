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

import "go.uber.org/multierr"

type RootCfg struct {
	Verbose  uint8
	Devel    bool
	RootPath string
}

type RootCfgBuilder struct {
	cfg *RootCfg

	Err error
}

// NewRootCfg creates RootCfg builder.
func NewRootCfg() *RootCfgBuilder {
	return &RootCfgBuilder{
		Err: nil,
		cfg: &RootCfg{},
	}
}

func (b *RootCfgBuilder) Build() (*RootCfg, error) {
	if b.Err != nil {
		return nil, b.Err
	}
	return &(*b.cfg), nil
}

func (b *RootCfgBuilder) AddErr(err error) {
	b.Err = multierr.Append(b.Err, err)
}

// Verbose sets verbose parameter to RootCfg.
func (b *RootCfgBuilder) Verbose(v uint8) *RootCfgBuilder {
	if v > 3 {
		b.AddErr(&ErrConfigValue{"verbose must be less than 4"})
	}
	b.cfg.Verbose = v
	return b
}

// Devel sets devel parameter to RootCfg.
func (b *RootCfgBuilder) Devel(v bool) *RootCfgBuilder {
	b.cfg.Devel = v
	return b
}

// RootPath sets rootpath parameter to RootCfg.
func (b *RootCfgBuilder) RootPath(v string) *RootCfgBuilder {
	b.cfg.RootPath = v
	return b
}
