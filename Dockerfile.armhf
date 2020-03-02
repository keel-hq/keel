FROM node:lts-alpine as ui
WORKDIR /app
COPY ui /app
RUN yarn
RUN yarn run lint --no-fix
RUN yarn run build

FROM arm32v7/debian:buster
ADD ca-certificates.crt /etc/ssl/certs/
COPY cmd/keel/release/keel-linux-arm /bin/keel
COPY --from=ui /app/dist /www
VOLUME /data
ENV XDG_DATA_HOME /data

EXPOSE 9300
ENTRYPOINT ["/bin/keel"]