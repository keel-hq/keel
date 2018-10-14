FROM arm32v6/alpine:3.8
ADD ca-certificates.crt /etc/ssl/certs/
COPY cmd/keel/release/keel-linux-arm /bin/keel
ENTRYPOINT ["/bin/keel"]