FROM --platform=$BUILDPLATFORM golang:1.23.4 as go-build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
COPY . /go/src/github.com/keel-hq/keel
WORKDIR /go/src/github.com/keel-hq/keel
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} make install

FROM --platform=$BUILDPLATFORM node:16.20.2-alpine as yarn-build
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
ENV XDG_DATA_HOME /data

ENTRYPOINT ["/bin/keel"]
EXPOSE 9300
