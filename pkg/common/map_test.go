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

package common_test

import (
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMergeMap(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"b": 3, "c": 4, "d": 5}
	m3 := map[string]int{"b": 6, "c": 7, "e": 8}
	want := map[string]int{"a": 1, "b": 6, "c": 7, "d": 5, "e": 8}
	assert.Equal(t, want, common.MergeMap(m1, m2, m3))
}

func TestSortedMapKeys(t *testing.T) {
	m := map[string]int{"b": 2, "c": 3, "a": 1}
	want := []string{"a", "b", "c"}
	assert.Equal(t, want, common.SortedMapKeys(m))
}
