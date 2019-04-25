FROM golang:1.12.0
COPY . /go/src/github.com/keel-hq/keel
WORKDIR /go/src/github.com/keel-hq/keel
RUN make build

FROM debian:latest
RUN apt-get update && apt-get install -y \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/*

COPY --from=0 /go/src/github.com/keel-hq/keel/cmd/keel/keel /bin/keel
ENTRYPOINT ["/bin/keel"]

EXPOSE 9300