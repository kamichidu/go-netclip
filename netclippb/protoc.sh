#!/bin/bash

set -e -u

__basedir="$(dirname "${0}")"
cd "${__basedir}"

if compgen -G *.go >/dev/null; then
    rm *.go
fi

protoc \
    --proto_path='.' \
    --proto_path='../.local/opt/googleapis' \
    --proto_path='../.local/opt/protoc-gen-validate' \
    --go_opt='paths=source_relative' \
    --go_out='.' \
    --go-grpc_opt='paths=source_relative' \
    --go-grpc_out='.' \
    --validate_opt='lang=go,paths=source_relative' \
    --validate_out='.' \
    *.proto
