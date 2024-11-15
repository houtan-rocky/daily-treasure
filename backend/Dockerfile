# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install necessary build tools
RUN apk add --no-cache git protoc protobuf-dev netcat-openbsd

# Install protoc-gen-go and protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# Make sure the protoc plugins are in PATH
ENV PATH="$PATH:$(go env GOPATH)/bin"

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Create proto output directory
RUN mkdir -p proto

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
