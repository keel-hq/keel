FROM golang:1.23.4-alpine AS go-build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
COPY . /go/src/github.com/keel-hq/keel
WORKDIR /go/src/github.com/keel-hq/keel

# Install build dependencies for musl-based static compilation
RUN apk add --no-cache git build-base musl-dev binutils-gold

# Build with CGO support for sqlite using musl - native build per platform
RUN git config --global --add safe.directory /go/src/github.com/keel-hq/keel && \
    GIT_REVISION=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev") && \
    JOBDATE=$(date -u +%Y-%m-%dT%H%M%SZ) && \
    CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -a -tags netgo \
    -ldflags "-w -s -linkmode external -extldflags '-static' -X github.com/keel-hq/keel/version.Version=${VERSION} -X github.com/keel-hq/keel/version.Revision=${GIT_REVISION} -X github.com/keel-hq/keel/version.BuildDate=${JOBDATE}" \
    -o /go/bin/keel ./cmd/keel

ARG BUILDPLATFORM
FROM --platform=$BUILDPLATFORM node:16.20.2-alpine AS yarn-build
WORKDIR /app
COPY ui /app
RUN yarn
RUN yarn run lint --no-fix
RUN yarn run build

FROM alpine:3.20.3
ARG USERNAME=keel
ARG USER_ID=666
ARG GROUP_ID=$USER_ID
ARG TARGETARCH

RUN apk --no-cache add ca-certificates
RUN addgroup --gid $GROUP_ID $USERNAME \
    && adduser --home /data --ingroup $USERNAME --disabled-password --uid $USER_ID $USERNAME \
    && mkdir -p /data && chown $USERNAME:0 /data && chmod g=u /data

COPY --from=go-build /go/bin/keel /bin/keel
COPY --from=yarn-build /app/dist /www

USER $USER_ID

VOLUME /data
ENV XDG_DATA_HOME=/data

ENTRYPOINT ["/bin/keel"]
EXPOSE 9300
