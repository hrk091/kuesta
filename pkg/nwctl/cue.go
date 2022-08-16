package nwctl

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"fmt"
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
		return cue.Value{}, v.Err()
	}
	return v, nil
}

// NewValueWithInstance creates cue.Value from cue build.Instance to resolve dependent imports.
func NewValueWithInstance(cctx *cue.Context, entrypoints []string, loadcfg *load.Config) (cue.Value, error) {
	if len(entrypoints) == 0 {
		return cue.Value{}, fmt.Errorf("no entrypoint files")
	}
	bis := load.Instances(entrypoints, loadcfg)
	if len(bis) != 1 {
		return cue.Value{}, fmt.Errorf("unexpected length of load.Instances result: %d", len(bis))
	}

	bi := bis[0]
	if bi.Err != nil {
		return cue.Value{}, bi.Err
	}
	v := cctx.BuildInstance(bi)
	if v.Err() != nil {
		return cue.Value{}, v.Err()
	}
	return v, nil
}

// ApplyTransform performs cue evaluation using given input and transform file.
// It returns cue.Iterator which iterates items including device name label and device config cue.Value.
func ApplyTransform(cctx *cue.Context, in cue.Value, transform cue.Value) (*cue.Iterator, error) {
	template := cctx.CompileString(CueSrcStrTemplate, cue.Scope(transform))
	if template.Err() != nil {
		return nil, template.Err()
	}
	filled := template.FillPath(cue.ParsePath(CuePathInput), in)
	if filled.Err() != nil {
		return nil, filled.Err()
	}
	filledIn := filled.LookupPath(cue.ParsePath(CuePathOutput))
	if err := filledIn.Validate(cue.Concrete(true)); err != nil {
		return nil, err
	}

	out := filled.LookupPath(cue.ParsePath(CuePathOutput)).Eval()
	if out.Err() != nil {
		return nil, out.Err()
	}
	it, err := out.LookupPath(cue.ParsePath(CuePathDevice)).Fields()
	if err != nil {
		return nil, err
	}
	return it, nil
}

// ExtractDeviceConfig extracts the device config from computed results of service transform apply.
func ExtractDeviceConfig(v cue.Value) ([]byte, error) {
	cfg := v.LookupPath(cue.ParsePath(CuePathConfig))
	if cfg.Err() != nil {
		return nil, cfg.Err()
	}
	syn := cfg.Syntax(cue.Final())
	return format.Node(syn)
}
