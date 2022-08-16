package nwctl

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"fmt"
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
