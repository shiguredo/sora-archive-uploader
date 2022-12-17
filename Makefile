VERSION := 2022.1.0
REVISION := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := "-X main.version=$(VERSION) -X main.revision=$(REVISION) -X main.buildDate=$(BUILD_DATE)"
LDFLAGS_PROD := "-s -w -X main.version=$(VERSION) -X main.revision=$(REVISION)"

export GO1111MODULE=on
export CWD=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: all sora-archive-uploader-dev sora-archive-uploader-prod
all: sora-archive-uploader-dev

sora-archive-uploader-dev: cmd/sora-archive-uploader/main.go
	go build -race -ldflags $(LDFLAGS) -o bin/$@ $<

sora-archive-uploader-prod: cmd/sora-archive-uploader/main.go
	go build -ldflags $(LDFLAGS_PROD) -o bin/$@ $<

test:
	go test -v ./db/test
