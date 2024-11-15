# Stage 1: Build Proto Descriptors
FROM alpine:3.15.1 AS build

WORKDIR /app

# Install protoc in the build stage
RUN apk add --no-cache protoc

# Copy the proto files into the build container
COPY ./proto/dailytreasure.proto .

# Build proto descriptor set (optional but useful for Envoy configuration)
RUN protoc --include_imports --include_source_info \
    --descriptor_set_out=dailytreasure.pb dailytreasure.proto

# Stage 2: Envoy Runtime
FROM envoyproxy/envoy:v1.27.0

# Copy the Envoy configuration file
COPY envoy.yaml /etc/envoy/envoy.yaml

# Copy the proto descriptor from the build stage
COPY --from=build /app/dailytreasure.pb /tmp/

# Start Envoy with the configuration
CMD ["/usr/local/bin/envoy", "-c", "/etc/envoy/envoy.yaml", "-l", "info"]
