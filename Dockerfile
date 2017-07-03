FROM golang:1.8.3 as builder
COPY . /go/src/github.com/rusenask/keel
WORKDIR /go/src/github.com/rusenask/keel
RUN go get && make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /go/src/github.com/rusenask/keel/keel /bin/keel
ENTRYPOINT ["/bin/keel"]

EXPOSE 9300