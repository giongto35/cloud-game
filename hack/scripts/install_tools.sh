#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

echo 'Installing go tools'

go get -mod='' -u golang.org/x/tools/cmd/goimports
go get -mod='' -u github.com/onsi/ginkgo/ginkgo
go get -mod='' -u github.com/vektra/mockery
go get -mod='' github.com/golangci/golangci-lint/cmd/golangci-lint
go get -mod='' -u golang.org/x/tools/cmd/cover
go get -mod='' -u github.com/mattn/goveralls
go get -mod='' -u honnef.co/go/tools/cmd/staticcheck
