FROM golang:1.12.0
COPY . /go/src/github.com/keel-hq/keel
WORKDIR /go/src/github.com/keel-hq/keel
RUN make install

FROM node:9.11.1-alpine
WORKDIR /app
COPY ui /app
RUN yarn
RUN yarn run lint --no-fix
RUN yarn run build

FROM alpine:latest
RUN apk --no-cache add ca-certificates

VOLUME /data
ENV XDG_DATA_HOME /data

COPY --from=0 /go/bin/keel /bin/keel
COPY --from=1 /app/dist /www
ENTRYPOINT ["/bin/keel"]
EXPOSE 9300