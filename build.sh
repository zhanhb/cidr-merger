#!/bin/sh
set -e
LDFLAGS="-X main.VERSION=$VERSION -s -w"
GCFLAGS=""
GOMIPS=softfloat

env CGO_ENABLED=0 GOOS=linux GOARCH=mips GOMIPS=$GOMIPS go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o cidr-merger

upx --best --ultra-brute cidr-merger
