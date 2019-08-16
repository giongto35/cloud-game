#!/usr/bin/env bash

set -e
touch coverage.txt

# test fuzz inputs
go test -tags gofuzz -run TestFuzz -v .

# quick-test without -race
go test ./...

# quick-test with "debug"
go test -tags debug ./...

for d in $(go list ./... | grep -v vendor); do
    go test -race -coverprofile=profile.out -covermode=atomic "$d"
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
