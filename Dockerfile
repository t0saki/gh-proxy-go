FROM golang:1.22-alpine AS builder
RUN go env -w GO111MODULE=auto \
  && go env -w CGO_ENABLED=0
#  && go env -w GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY ./ .

RUN set -ex \
    && cd /build \
    && go build -ldflags "-s -w -extldflags '-static'" -o gh-proxy

FROM alpine:latest

COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh && \
    apk add --no-cache --update \
      coreutils \
      shadow \
      su-exec \
      tzdata && \
    rm -rf /var/cache/apk/* && \
    mkdir -p /app && \
    mkdir -p /config && \
    useradd -d /config -s /bin/sh abc && \
    chown -R abc /config

ENV TZ="Asia/Shanghai"
ENV UID=99
ENV GID=100
ENV UMASK=002
ENV WHITE_LIST=""
ENV BLACK_LIST="\"example3\",\"example4\""
ENV ALLOW_PROXY_ALL="false"
ENV OTHER_WHITE_LIST=""
ENV OTHER_BLACK_LIST="\"example3\",\"example4\""
ENV HTTP_HOST=""
ENV HTTP_PORT="8080"
ENV SIZE_LIMIT="10737418240"

COPY --from=builder /build/gh-proxy /app/
COPY --from=builder /build/config.json.dist /app/

WORKDIR /app

VOLUME [ "/app" ]

ENTRYPOINT [ "/docker-entrypoint.sh" ]