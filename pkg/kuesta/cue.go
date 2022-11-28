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
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/token"
	cuejson "cuelang.org/go/encoding/json"
	"fmt"
	"github.com/nttcom/kuesta/pkg/common"
	"github.com/pkg/errors"
	"strconv"
)

// NewValueFromBytes creates cue.Value from given []byte.
func NewValueFromBytes(cctx *cue.Context, buf []byte) (cue.Value, error) {
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
	case cue.NumberKind:
		return func(s string) (any, error) {
			return strconv.ParseFloat(s, 64)
		}, nil
	case cue.BoolKind:
		return func(s string) (any, error) {
			return strconv.ParseBool(s)
		}, nil
	case cue.NullKind:
		return func(s string) (any, error) {
			return nil, nil
		}, nil
	default:
		err := fmt.Errorf("unexpected kind: %s", kind)
		return func(s string) (any, error) {
			return s, nil
		}, errors.WithStack(err)
	}
}
