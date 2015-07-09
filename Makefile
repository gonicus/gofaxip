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
	rm -rf packages
	rm -f gofaxbuild.id

deb: iid := $(shell cat /proc/sys/kernel/random/uuid)
deb:
	docker build -t $(iid) -f Dockerfile.debbuilder .
	docker run --cidfile=gofaxbuild.id $(iid)
	docker cp `cat gofaxbuild.id`:/usr/src/build/packages .
	docker rm `cat gofaxbuild.id`
	docker rmi $(iid)
	rm gofaxbuild.id
