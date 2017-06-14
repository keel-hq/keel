FROM golang:1.8.1-alpine
COPY . /go/src/github.com/rusenask/keel
WORKDIR /go/src/github.com/rusenask/keel
RUN apk add --no-cache git && go get
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo  -ldflags  -'w' -o keel .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=0 /go/src/github.com/rusenask/keel/keel /bin/keel
ENTRYPOINT ["/bin/keel"]

EXPOSE 9300