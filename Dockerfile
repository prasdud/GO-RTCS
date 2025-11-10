# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY server/ ./server/
WORKDIR /app/server
RUN go build -o server main.go

# Run stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server/server .
EXPOSE 8080
CMD ["./server"]
