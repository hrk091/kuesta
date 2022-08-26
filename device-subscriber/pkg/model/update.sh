#!/bin/bash
set -o nounset
set -o xtrace

go install github.com/openconfig/ygot/generator@latest
generator -generate_fakeroot -output_file generated.go -package_name model -path=yang -compress_paths=true -shorten_enum_leaf_names -typedef_enum_with_defmod -exclude_modules=ietf-interfaces yang/openconfig-interfaces.yang yang/openconfig-if-ip.yang
