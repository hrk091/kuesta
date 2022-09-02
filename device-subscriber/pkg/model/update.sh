#!/bin/bash
#
# Copyright 2022 NTT Communications Corporation.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -o nounset
set -o xtrace

go install github.com/openconfig/ygot/generator@latest
generator -generate_fakeroot -output_file generated.go -package_name model -path=yang -compress_paths=true -shorten_enum_leaf_names -typedef_enum_with_defmod -exclude_modules=ietf-interfaces yang/openconfig-interfaces.yang yang/openconfig-if-ip.yang
