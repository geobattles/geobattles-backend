FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf

RUN apk add --no-cache bash git make openssh-client ca-certificates \
    && addgroup -g 1000 dev \
    && adduser -D -u 1000 -G dev dev \
    && mkdir -p /home/dev/go /home/dev/.cache/go-build \
    && chown -R dev:dev /home/dev

ENV HOME=/home/dev \
    GOPATH=/home/dev/go \
    GOTOOLCHAIN=local \
    GOWORK=off \
    PATH=/home/dev/go/bin:$PATH

USER dev
