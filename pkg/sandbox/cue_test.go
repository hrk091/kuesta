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

package sandbox_test

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/json"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"runtime/debug"
	"testing"
)

func exitOnErr(t *testing.T, err error) {
	if err != nil {
		t.Log(string(debug.Stack()))
		t.Fatal(err)
	}
}

func TestCueExtract(t *testing.T) {
	t.Skip()
	jsonVal := []byte(`{"port": 2, "desc": "test"}`)
	cctx := cuecontext.New()
	expr, err := json.Extract("test", jsonVal)
	exitOnErr(t, err)
	v := cctx.BuildExpr(expr)
	t.Fatal(v)
}

func TestCueTypeExtract(t *testing.T) {
	given := []byte(`#Input: {
	device: string
	port:   uint16
	noShut: bool
	desc:   string | *""
	mtu:    uint16 | *9000
}
`)
	cctx := cuecontext.New()
	val, err := nwctl.NewValueFromBytes(cctx, given)
	exitOnErr(t, err)

	inputVal := val.LookupPath(cue.ParsePath("#Input"))
	portVal := inputVal.LookupPath(cue.ParsePath("port"))
	deviceVal := inputVal.LookupPath(cue.ParsePath("device"))
	descVal := inputVal.LookupPath(cue.ParsePath("desc"))

	assert.Equal(t, cue.StringKind, deviceVal.IncompleteKind())
	assert.Equal(t, cue.StringKind, descVal.IncompleteKind())
	assert.Equal(t, cue.IntKind, portVal.IncompleteKind())
}
