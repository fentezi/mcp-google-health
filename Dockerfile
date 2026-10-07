FROM golang:1.26.6 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/server ./cmd/main

FROM alpine:latest

WORKDIR /app

COPY --from=builder /bin/server ./

ENV SERVER_HOST=0.0.0.0 \
    OAUTH_TOKEN_FILE=/data/token.json

VOLUME /data

EXPOSE 8080 8060

CMD ["./server"]
