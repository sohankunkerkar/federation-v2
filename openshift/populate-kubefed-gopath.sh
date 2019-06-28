#!/bin/bash -eu

# This script populates the federation-v2 directory within a gopath directory
# structure with the vendored federation-v2 source from this repo. It can be run
# from any directory. The gopath directory structure is assumed to be rooted at
# '/go' but can be overriden by setting the GOPATH_DIR environment variable.

dir=$(realpath "$(dirname "${BASH_SOURCE}")/..")
gopath_dir="${GOPATH_DIR:-"/go"}"

kubefed_gp=sigs.k8s.io/kubefed
kubefed_src="${gopath_dir}/src/${kubefed_gp}"

echo "Populating kubefed gopath at ${kubefed_src}"
mkdir -p $kubefed_src
rsync Makefile $kubefed_src/Makefile
rsync -r pkg $kubefed_src/pkg
rsync -r cmd $kubefed_src/cmd
rsync -r test $kubefed_src/test
rsync -r hack $kubefed_src/hack
rsync -r scripts $kubefed_src/scripts
rsync -r config $kubefed_src/config
rsync -r vendor $kubefed_src/vendor
