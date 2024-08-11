#!/bin/bash
#
env CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-X main.version=$1 -X main.buildTime="$(date '+%Y%m%d_%H:%M:%S')" -extldflags=-static -w -s" -o gozstd-linux-amd64 play/working/main.go
env CGO_ENABLED=0 GOOS=windows go build -trimpath -ldflags="-X main.version=$1 -X main.buildTime="$(date '+%Y%m%d_%H:%M:%S')" -extldflags=-static -w -s" -o gozstd-windows-amd64.exe play/working/main.go

