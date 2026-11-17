FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN go mod verify && CGO_ENABLED=0 GOTOOLCHAIN=local go build -mod=readonly -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o ./out/go-geo-server

FROM alpine:3.23@sha256:469c9ea410b9215eaee7976ba6c0639039b444be8e5d638f7abbada189971eb2

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/out/go-geo-server /app/go-geo-server
COPY --from=builder /app/assets /app/assets

USER 65532:65532

EXPOSE 8080

CMD [ "/app/go-geo-server" ]