GO=go
VERSION=git-$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X main.version '$(VERSION)'"

GOPATH := $(shell pwd)
export GOPATH

all: bin/gofaxsend bin/gofaxd

bin/gofaxd:
	$(GO) install $(LDFLAGS) gofaxd

bin/gofaxsend:
	$(GO) install $(LDFLAGS) gofaxsend

clean:
	rm -rf bin
