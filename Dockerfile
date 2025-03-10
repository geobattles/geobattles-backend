FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o ./out/go-geo-server

FROM alpine:latest

COPY --from=builder /app/out/go-geo-server /app/go-geo-server
COPY --from=builder /app/assets /assets

EXPOSE 8080

CMD [ "/app/go-geo-server" ]