# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install necessary build tools
RUN apk add --no-cache git protoc

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Generate protobuf files
RUN protoc -I=proto \
    --go_out=proto \
    --go_opt=paths=source_relative \
    --go-grpc_out=proto \
    --go-grpc_opt=paths=source_relative \
    proto/dailytreasure.proto

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/app server/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /go/bin/app /app/app

# Expose the gRPC port
EXPOSE 50051

# Run the application
CMD ["/app/app"]