FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go-build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG VERSION
ARG REVISION
ARG BUILD_DATE
COPY . /go/src/github.com/keel-hq/keel
WORKDIR /go/src/github.com/keel-hq/keel

# Install build dependencies for musl-based static compilation
RUN apk add --no-cache git build-base musl-dev binutils-gold

# Build with CGO support for sqlite using musl - native build per platform
RUN git config --global --add safe.directory /go/src/github.com/keel-hq/keel && \
    EFFECTIVE_REVISION=${REVISION:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")} && \
    EFFECTIVE_VERSION=${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")} && \
    EFFECTIVE_BUILD_DATE=${BUILD_DATE:-$(date -u +%Y-%m-%dT%H%M%SZ)} && \
    CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -a -tags netgo \
    -ldflags "-w -s -linkmode external -extldflags '-static' -X github.com/keel-hq/keel/version.Version=${EFFECTIVE_VERSION} -X github.com/keel-hq/keel/version.Revision=${EFFECTIVE_REVISION} -X github.com/keel-hq/keel/version.BuildDate=${EFFECTIVE_BUILD_DATE}" \
    -o /go/bin/keel ./cmd/keel

ARG BUILDPLATFORM
FROM --platform=$BUILDPLATFORM node:24.11.0-alpine@sha256:f36fed0b2129a8492535e2853c64fbdbd2d29dc1219ee3217023ca48aebd3787 AS ui-build
WORKDIR /app
COPY ui/package.json ui/package-lock.json /app/
RUN npm ci
COPY ui /app
RUN npm run typecheck && npm run lint && npm test && npm run build

FROM alpine:3.20.3@sha256:1e42bbe2508154c9126d48c2b8a75420c3544343bf86fd041fb7527e017a4b4a
ARG USERNAME=keel
ARG USER_ID=666
ARG GROUP_ID=$USER_ID
ARG TARGETARCH

RUN apk --no-cache add ca-certificates
RUN addgroup --gid $GROUP_ID $USERNAME \
    && adduser --home /data --ingroup $USERNAME --disabled-password --uid $USER_ID $USERNAME \
    && mkdir -p /data /run/secrets/kubernetes.io \
    && chown $USERNAME:0 /data \
    && chmod g=u /data \
    && chmod 0755 /run/secrets /run/secrets/kubernetes.io

COPY --from=go-build /go/bin/keel /bin/keel
COPY --from=ui-build /app/dist /www

USER $USER_ID

VOLUME /data
ENV XDG_DATA_HOME=/data

ENTRYPOINT ["/bin/keel"]
EXPOSE 9300
