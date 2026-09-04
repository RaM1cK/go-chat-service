FROM golang:1.26.4-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/chat-service ./cmd/main.go

FROM alpine:3.19

WORKDIR /chat-service

COPY --from=builder /bin/chat-service .
COPY migrations/ ./migrations/

CMD ["./chat-service"]
