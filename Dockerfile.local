FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY       keel /bin/keel
ENTRYPOINT ["/bin/keel"]

EXPOSE 9300