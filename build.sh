#!/bin/sh
set -e

VERSION="$1"
if [[ -z "$VERSION" ]]; then
    VERSION=`git describe --exact-match --tags 2>/dev/null | sed 's/^v//'`
    if [[ -z "$VERSION" ]]; then
        VERSION="`git rev-list -1 HEAD`"
    fi
fi
LDFLAGS="-X main.VERSION=$VERSION -s -w"
GCFLAGS=""
GOMIPS=softfloat

env CGO_ENABLED=0 GOOS=linux GOARCH=mips "GOMIPS=$GOMIPS" go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o cidr-merger

upx --best --ultra-brute cidr-merger
