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
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/token"
	cuejson "cuelang.org/go/encoding/json"
	"fmt"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/pkg/errors"
	"strconv"
)

var (
	cueTypeStrInput    = "#Input"
	cueTypeStrTemplate = "#Template"
	cuePathInput       = "input"
	cuePathOutput      = "output"
	cuePathDevice      = "devices"
	cuePathConfig      = "config"
)

// NewValueFromBuf creates cue.Value from given []byte.
func NewValueFromBuf(cctx *cue.Context, buf []byte) (cue.Value, error) {
	v := cctx.CompileBytes(buf)
	if v.Err() != nil {
		return cue.Value{}, errors.WithStack(v.Err())
	}
	return v, nil
}

// NewValueFromJson creates cue.Value from the given JSON []byte.
//
// Deprecated: cuejson.Extract causes decoding to wrong type. Use json.UnMarshal to map[string]any and NewAstExpr instead.
func NewValueFromJson(cctx *cue.Context, buf []byte) (cue.Value, error) {
	expr, err := cuejson.Extract("from json", buf)
	if err != nil {
		return cue.Value{}, errors.WithStack(fmt.Errorf("extract JSON: %w", err))
	}
	v := cctx.BuildExpr(expr)
	if v.Err() != nil {
		return cue.Value{}, errors.WithStack(fmt.Errorf("build cue.Value from expr: %w", v.Err()))
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
	template := cctx.CompileString(cueTypeStrTemplate, cue.Scope(transform))
	if template.Err() != nil {
		return nil, errors.WithStack(template.Err())
	}
	filled := template.FillPath(cue.ParsePath(cuePathInput), in)
	if filled.Err() != nil {
		return nil, errors.WithStack(filled.Err())
	}
	filledIn := filled.LookupPath(cue.ParsePath(cuePathOutput))
	if err := filledIn.Validate(cue.Concrete(true)); err != nil {
		return nil, errors.WithStack(err)
	}

	out := filled.LookupPath(cue.ParsePath(cuePathOutput)).Eval()
	if out.Err() != nil {
		return nil, errors.WithStack(out.Err())
	}
	it, err := out.LookupPath(cue.ParsePath(cuePathDevice)).Fields()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return it, nil
}

// ExtractDeviceConfig extracts the device config from computed results of service transform apply.
func ExtractDeviceConfig(v cue.Value) ([]byte, error) {
	cfg := v.LookupPath(cue.ParsePath(cuePathConfig))
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

// NewAstExpr returns CUE AST Expression for the given value.
func NewAstExpr(value any) ast.Expr {
	switch val := value.(type) {
	case nil:
		return ast.NewNull()
	case bool:
		return ast.NewBool(val)
	case string:
		return ast.NewString(val)
	case float64, int:
		// json decoder always parses number as float64
		// and some yaml decoder parses number as int
		return newAstNumber(val)
	case []any:
		var items []ast.Expr
		for _, item := range val {
			items = append(items, NewAstExpr(item))
		}
		return ast.NewList(items...)
	case map[string]any:
		var fields []any
		for _, k := range common.SortedMapKeys(val) {
			v := val[k]
			key := ast.NewIdent(k)
			value := NewAstExpr(v)
			f := &ast.Field{
				Label: key,
				Value: value,
			}
			fields = append(fields, f)
		}
		return ast.NewStruct(fields...)
	case map[any]any:
		var fields []any
		// not sorted since there are no general way to compare two keys
		for k, v := range val {
			key := ast.NewIdent(k.(string))
			value := NewAstExpr(v)
			f := &ast.Field{
				Label: key,
				Value: value,
			}
			fields = append(fields, f)
		}
		return ast.NewStruct(fields...)
	}
	return &ast.BottomLit{}
}

// newAstNumber resolves CUE Integer or Float value.
func newAstNumber(n any) *ast.BasicLit {
	str := fmt.Sprintf("%v", n)
	if _, err := strconv.ParseInt(str, 0, 64); err == nil {
		return &ast.BasicLit{Kind: token.INT, Value: str}
	}
	return &ast.BasicLit{Kind: token.FLOAT, Value: str}
}

// CueKindOf returns cue.Kind of the value placed at the given path.
func CueKindOf(v cue.Value, path string) cue.Kind {
	if path == "" {
		return v.IncompleteKind()
	} else {
		return v.LookupPath(cue.ParsePath(path)).IncompleteKind()
	}
}

type StrConvFunc func(string) (any, error)

// NewStrConvFunc returns StrConvFunc to convert string to the corresponding type of the given cue.Kind.
func NewStrConvFunc(kind cue.Kind) (StrConvFunc, error) {
	switch kind {
	case cue.StringKind:
		return func(s string) (any, error) {
			return s, nil
		}, nil
	case cue.IntKind:
		return func(s string) (any, error) {
			return strconv.Atoi(s)
		}, nil
	case cue.FloatKind:
		return func(s string) (any, error) {
			return strconv.ParseFloat(s, 64)
		}, nil
	case cue.BoolKind:
		return func(s string) (any, error) {
			return strconv.ParseBool(s)
		}, nil
	default:
		err := fmt.Errorf("unexpected kind: %s", kind)
		return func(s string) (any, error) {
			return s, nil
		}, errors.WithStack(err)
	}
}

func ConvertInputKeys(transformVal cue.Value, keys map[string]string) (map[string]any, error) {
	converted := map[string]any{}
	for k, v := range keys {
		kind := CueKindOf(transformVal, fmt.Sprintf("%s.%s", cueTypeStrInput, k))
		convert, err := NewStrConvFunc(kind)
		if err != nil {
			return nil, fmt.Errorf("the type of primary key=%s must be string|int|float|bool: %w", k, err)
		}
		vv, err := convert(v)
		if err != nil {
			return nil, fmt.Errorf("type mismatch: key=%s, value=%s: %w", k, v, err)
		}
		converted[k] = vv
	}
	return converted, nil
}
