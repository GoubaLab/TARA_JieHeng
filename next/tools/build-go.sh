#!/bin/sh
set -eu
project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
mkdir -p "$project_dir/next/bin"
cd "$project_dir/next/server"
go build -trimpath -o "$project_dir/next/bin/jieheng" ./cmd/jieheng
go env -json GOOS GOARCH GOVERSION > "$project_dir/next/bin/build-info.json"
