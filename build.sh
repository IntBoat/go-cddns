#!/bin/sh

# STEP 1: Determinate the required values
PACKAGE="$(head -1 go.mod | sed 's/module\s//')"
VERSION="$(git describe --tags --always --abbrev=0 --match='v[0-9]*.[0-9]*.[0-9]*' 2> /dev/null | sed 's/^.//')"
COMMIT_HASH="$(git rev-parse --short HEAD)"
BUILD_TIMESTAMP=$(date '+%Y-%m-%dT%H:%M:%S')

# STEP 2: Build the ldflags
LDFLAGS="-X '${PACKAGE}/version.Version=${VERSION}'"
LDFLAGS="${LDFLAGS} -X '${PACKAGE}/version.CommitHash=${COMMIT_HASH}'"
LDFLAGS="${LDFLAGS} -X '${PACKAGE}/version.BuildTime=${BUILD_TIMESTAMP}'"

echo $LDFLAGS
# STEP 3: Actual Go build process
CGO_ENABLED=0
go build -ldflags "-s -w ${LDFLAGS}" -o tmp
upx -q --lzma tmp -o server