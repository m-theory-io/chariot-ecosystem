#!/bin/bash
# Script to build mac and linux versions of project to ./bin
# Args: 
#   1 - name of the native executable (defaults to "chariot" if not provided)

name=${1:-"chariot"}
output="./bin/"
lsuffix="-linux"
msuffix="-mac"
wsuffix="-windows"
native=$output$name
linux=$output$name$lsuffix
mac=$output$name$msuffix
windows=$output$name$wsuffix

# Test for existence of output directory
if [ ! -d "$output" ]; then
    # create directory
    echo creating directory $output
    mkdir -p $output
fi

echo "Building $native"
go build -o $native ./cmd

echo "Building $linux"
env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $linux ./cmd

echo "Building $mac"
env GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o $mac ./cmd

echo "Building $windows"
env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $windows ./cmd