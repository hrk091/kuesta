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
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"fmt"
	"github.com/pkg/errors"
)

var (
	CueSrcStrTemplate = "#Template"
	CuePathInput      = "input"
	CuePathOutput     = "output"
	CuePathDevice     = "devices"
	CuePathConfig     = "config"
)

// NewValueFromBuf creates cue.Value from given []byte.
func NewValueFromBuf(cctx *cue.Context, buf []byte) (cue.Value, error) {
	v := cctx.CompileBytes(buf)
	if v.Err() != nil {
		return cue.Value{}, errors.WithStack(v.Err())
	}
	return v, nil
}

// NewValueWithInstance creates cue.Value from cue build.Instance to resolve dependent imports.
func NewValueWithInstance(cctx *cue.Context, entrypoints []string, loadcfg *load.Config) (cue.Value, error) {
	if len(entrypoints) == 0 {
		return cue.Value{}, errors.WithStack(fmt.Errorf("no entrypoint files"))
	}
	bis := load.Instances(entrypoints, loadcfg)
	if len(bis) != 1 {
		return cue.Value{}, errors.WithStack(fmt.Errorf("unexpected length of load.Instances result: %d", len(bis)))
	}

	bi := bis[0]
	if bi.Err != nil {
		return cue.Value{}, errors.WithStack(bi.Err)
	}
	v := cctx.BuildInstance(bi)
	if v.Err() != nil {
		return cue.Value{}, errors.WithStack(v.Err())
	}
	return v, nil
}

// ApplyTransform performs cue evaluation using given input and transform file.
// It returns cue.Iterator which iterates items including device name label and device config cue.Value.
func ApplyTransform(cctx *cue.Context, in cue.Value, transform cue.Value) (*cue.Iterator, error) {
	template := cctx.CompileString(CueSrcStrTemplate, cue.Scope(transform))
	if template.Err() != nil {
		return nil, errors.WithStack(template.Err())
	}
	filled := template.FillPath(cue.ParsePath(CuePathInput), in)
	if filled.Err() != nil {
		return nil, errors.WithStack(filled.Err())
	}
	filledIn := filled.LookupPath(cue.ParsePath(CuePathOutput))
	if err := filledIn.Validate(cue.Concrete(true)); err != nil {
		return nil, errors.WithStack(err)
	}

	out := filled.LookupPath(cue.ParsePath(CuePathOutput)).Eval()
	if out.Err() != nil {
		return nil, errors.WithStack(out.Err())
	}
	it, err := out.LookupPath(cue.ParsePath(CuePathDevice)).Fields()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return it, nil
}

// ExtractDeviceConfig extracts the device config from computed results of service transform apply.
func ExtractDeviceConfig(v cue.Value) ([]byte, error) {
	cfg := v.LookupPath(cue.ParsePath(CuePathConfig))
	if cfg.Err() != nil {
		return nil, errors.WithStack(cfg.Err())
	}
	return FormatCue(cfg, cue.Final())
}

// FormatCue formats cue.Value in canonical cue fmt style.
func FormatCue(v cue.Value, opts ...cue.Option) ([]byte, error) {
	syn := v.Syntax(opts...)
	return format.Node(syn)
}
