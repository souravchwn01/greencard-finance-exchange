FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/server ./server
COPY --from=builder /app/data ./data

ENV APP_PORT=8080
EXPOSE ${APP_PORT}

CMD ["./server"]
