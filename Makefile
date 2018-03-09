JOBDATE		?= $(shell date -u +%Y-%m-%dT%H%M%SZ)
GIT_REVISION	= $(shell git rev-parse --short HEAD)
VERSION		?= $(shell git describe --tags --abbrev=0)

LDFLAGS		+= -X github.com/keel-hq/keel/version.Version=$(VERSION)
LDFLAGS		+= -X github.com/keel-hq/keel/version.Revision=$(GIT_REVISION)
LDFLAGS		+= -X github.com/keel-hq/keel/version.BuildDate=$(JOBDATE)

.PHONY: release

test:
	go test -v `go list ./... | egrep -v /vendor/`

build:
	@echo "++ Building keel"
	CGO_ENABLED=0 GOOS=linux cd cmd/keel && go build -a -tags netgo -ldflags "$(LDFLAGS) -w -s" -o keel .

install:
	@echo "++ Installing keel"
	CGO_ENABLED=0 GOOS=linux go install -ldflags "$(LDFLAGS) -w -s" github.com/keel-hq/keel/cmd/keel

image:
	docker build -t keelhq/keel:alpha -f Dockerfile .

alpha: image
	@echo "++ Pushing keel alpha"	
	docker push keelhq/keel:alpha