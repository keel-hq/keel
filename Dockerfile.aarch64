FROM arm64v8/alpine:3.8
ADD ca-certificates.crt /etc/ssl/certs/
COPY cmd/keel/release/keel-linux-aarch64 /bin/keel
ENTRYPOINT ["/bin/keel"]