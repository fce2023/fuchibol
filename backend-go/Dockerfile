FROM golang:alpine AS builder

WORKDIR /app

# Copy source code
COPY . .

# Ensure all dependencies are downloaded and synced
RUN go get github.com/gofiber/websocket/v2
RUN go mod tidy

# Build the main API server
RUN CGO_ENABLED=0 GOOS=linux go build -o api_server main.go

# Build the background worker
RUN CGO_ENABLED=0 GOOS=linux go build -o worker_server cmd/worker/main.go

# Final minimal image
FROM alpine:latest
RUN apk add --no-cache ffmpeg
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/api_server .
COPY --from=builder /app/worker_server .

EXPOSE 8000

CMD ["./api_server"]
