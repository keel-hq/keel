JOBDATE		?= $(shell date -u +%Y-%m-%dT%H%M%SZ)
GIT_REVISION	= $(shell git rev-parse --short HEAD)
VERSION		?= $(shell git describe --tags --abbrev=0)

LDFLAGS		+= -linkmode external -extldflags -static
LDFLAGS		+= -X github.com/keel-hq/keel/version.Version=$(VERSION)
LDFLAGS		+= -X github.com/keel-hq/keel/version.Revision=$(GIT_REVISION)
LDFLAGS		+= -X github.com/keel-hq/keel/version.BuildDate=$(JOBDATE)

ARMFLAGS		+= -a -v
ARMFLAGS		+= -X github.com/keel-hq/keel/version.Version=$(VERSION)
ARMFLAGS		+= -X github.com/keel-hq/keel/version.Revision=$(GIT_REVISION)
ARMFLAGS		+= -X github.com/keel-hq/keel/version.BuildDate=$(JOBDATE)

.PHONY: release

fetch-certs:
	curl --remote-name --time-cond cacert.pem https://curl.haxx.se/ca/cacert.pem
	cp cacert.pem ca-certificates.crt

compress:
	upx --brute cmd/keel/release/keel-linux-arm
	upx --brute cmd/keel/release/keel-linux-aarch64

build-binaries:
	go get github.com/mitchellh/gox
	@echo "++ Building keel binaries"
	cd cmd/keel && CC=arm-linux-gnueabi-gcc gox -verbose -output="release/{{.Dir}}-{{.OS}}-{{.Arch}}" \
		-ldflags "$(LDFLAGS)" -osarch="linux/arm"

build-arm:
	cd cmd/keel && env CC=arm-linux-gnueabihf-gcc CGO_ENABLED=1 GOARCH=arm GOOS=linux go build -ldflags="$(ARMFLAGS)" -o release/keel-linux-arm
	# disabling for now 64bit builds
	# cd cmd/keel && env GOARCH=arm64 GOOS=linux go build -ldflags="$(ARMFLAGS)" -o release/keel-linux-aarc64

armhf-latest:
	docker build -t keelhq/keel-arm:latest -f Dockerfile.armhf .
	docker push keelhq/keel-arm:latest

aarch64-latest:
	docker build -t keelhq/keel-aarch64:latest -f Dockerfile.aarch64 .
	docker push keelhq/keel-aarch64:latest

armhf:
	docker build -t keelhq/keel-arm:$(VERSION) -f Dockerfile.armhf .
	# docker push keelhq/keel-arm:$(VERSION)

aarch64:
	docker build -t keelhq/keel-aarch64:$(VERSION) -f Dockerfile.aarch64 .
	docker push keelhq/keel-aarch64:$(VERSION)

arm: build-arm fetch-certs armhf aarch64

test:
	go install github.com/mfridman/tparse@latest
	go test -json -v `go list ./... | egrep -v /tests` -cover | tparse -all -smallscreen

build:
	@echo "++ Building keel"
	GOOS=linux cd cmd/keel && go build -a -tags netgo -ldflags "$(LDFLAGS) -w -s" -o keel .

install:
	@echo "++ Installing keel"
	# CGO_ENABLED=0 GOOS=linux go install -ldflags "$(LDFLAGS)" github.com/keel-hq/keel/cmd/keel	
	GOOS=linux go install -ldflags "$(LDFLAGS)" github.com/keel-hq/keel/cmd/keel	

image:
	docker build -t keelhq/keel:alpha -f Dockerfile .

image-debian:
	docker build -t keelhq/keel:alpha -f Dockerfile.debian .

alpha: image
	@echo "++ Pushing keel alpha"
	docker push keelhq/keel:alpha

e2e: install
	cd tests && go test

run:
	go install github.com/keel-hq/keel/cmd/keel
	keel --no-incluster --ui-dir ui/dist

lint-ui:
	cd ui && yarn 
	yarn run lint --no-fix && yarn run build

run-ui:
	cd ui && yarn run serve

build-ui:
	docker build -t keelhq/keel:ui -f Dockerfile .
	docker push keelhq/keel:ui

run-debug: install
	DEBUG=true keel --no-incluster