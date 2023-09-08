.PHONY: all test

all:
	go build -o bin/sora-archive-uploader cmd/sora-archive-uploader/main.go

test:
	go test -race -v ./s3