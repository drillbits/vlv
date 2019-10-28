#!/bin/bash

cd "$GITHUB_WORKSPACE"

which go
go version
go env

go build -o vlv ./cmd/vlv

./vlv help
