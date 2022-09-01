FROM golang:alpine AS builder

RUN apk add --no-cache git
WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o ./out/go-geo-server

FROM alpine:latest

COPY --from=builder /app/out/go-geo-server /app/go-geo-server
COPY --from=builder /app/assets /assets

EXPOSE 8080

CMD [ "/app/go-geo-server" ]