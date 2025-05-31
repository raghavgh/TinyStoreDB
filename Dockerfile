# Use minimal Go base image
FROM golang:1.24-alpine as builder

WORKDIR /app
COPY . .

# Build your TinyStoreDB binary
RUN go build -o tinystoredb main.go

# Final stage - use scratch or alpine
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/tinystore .

# Expose gRPC port
EXPOSE 50051

# Optional: allow custom args/env
ENV TINYSTOREDB_PORT=50051

# Run your DB engine
CMD ["./tinystoredb"]