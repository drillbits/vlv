#!/bin/bash

cd "$GITHUB_WORKSPACE"

which go
go version
go env

go build ./cmd/vlv
