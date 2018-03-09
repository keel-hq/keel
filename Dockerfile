FROM golang:1.9.3
COPY . /go/src/github.com/keel-hq/keel
WORKDIR /go/src/github.com/keel-hq/keel
RUN make install

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=0 /go/bin/keel /bin/keel
ENTRYPOINT ["/bin/keel"]

EXPOSE 9300