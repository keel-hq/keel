JOBDATE		?= $(shell date -u +%Y-%m-%dT%H%M%SZ)
GIT_REVISION	= $(shell git rev-parse --short HEAD)
VERSION		?= $(shell git describe --tags --abbrev=0)

SWAG_VERSION	= v1.16.6
SWAG		= go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
OPENAPI_GENERATOR_VERSION = v7.22.0
OPENAPI_GENERATOR_IMAGE = openapitools/openapi-generator-cli:$(OPENAPI_GENERATOR_VERSION)
SPECTRAL_VERSION = 6.16.1
SPECTRAL_IMAGE = stoplight/spectral:$(SPECTRAL_VERSION)
API_CLIENT_DIR = ui/src/api/generated

LDFLAGS		+= -linkmode external -extldflags -static
LDFLAGS		+= -X github.com/keel-hq/keel/version.Version=$(VERSION)
LDFLAGS		+= -X github.com/keel-hq/keel/version.Revision=$(GIT_REVISION)
LDFLAGS		+= -X github.com/keel-hq/keel/version.BuildDate=$(JOBDATE)

ARMFLAGS		+= -a -v
ARMFLAGS		+= -X github.com/keel-hq/keel/version.Version=$(VERSION)
ARMFLAGS		+= -X github.com/keel-hq/keel/version.Revision=$(GIT_REVISION)
ARMFLAGS		+= -X github.com/keel-hq/keel/version.BuildDate=$(JOBDATE)

.PHONY: release api-spec api-client api-generate api-validate api-check

api-spec:
	$(SWAG) init -g cmd/keel/main.go -d . --parseInternal --parseDependency --outputTypes yaml -o docs

api-client:
	mkdir -p $(API_CLIENT_DIR)
	find $(API_CLIENT_DIR) -mindepth 1 -delete
	cp docs/openapi-generator-ignore $(API_CLIENT_DIR)/.openapi-generator-ignore
	docker run --rm --user "$$(id -u):$$(id -g)" -v "$(CURDIR):/local" $(OPENAPI_GENERATOR_IMAGE) generate \
		-i /local/docs/swagger.yaml \
		-g javascript \
		-o /local/$(API_CLIENT_DIR) \
		-c /local/docs/openapi-generator-config.yaml
	sed -i "s|constructor(basePath = 'http://localhost')|constructor(basePath = '')|" $(API_CLIENT_DIR)/ApiClient.js
	find $(API_CLIENT_DIR)/.openapi-generator -depth -delete
	find $(API_CLIENT_DIR)/.openapi-generator-ignore -delete

api-generate: api-spec api-client

api-validate:
	docker run --rm -v "$(CURDIR):/local" $(SPECTRAL_IMAGE) lint /local/docs/swagger.yaml --ruleset /local/docs/spectral.yaml
	go test -v ./pkg/http -run TestOpenAPIContract

api-check: api-generate api-validate
	git diff --exit-code -- docs/swagger.yaml ui/src/api/generated

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

install-debug:
	@echo "++ Installing keel with debug flags"
	go install github.com/go-delve/delve/cmd/dlv@latest
	GOOS=linux go install -gcflags "all=-N -l" -ldflags "$(LDFLAGS)" github.com/keel-hq/keel/cmd/keel

image:
	docker build -t keelhq/keel:alpha -f Dockerfile .

image-debian:
	docker build -t keelhq/keel:alpha -f Dockerfile.debian .

alpha: image
	@echo "++ Pushing keel alpha"
	docker push keelhq/keel:alpha

e2e:
	./.test/e2e-k3s.sh

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
