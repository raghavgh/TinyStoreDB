FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

# Build server and CLI binaries
RUN go build -o tinystoredb-server main.go
RUN go build -o tinystoredb-cli ./cmd/tinystoredb-cli/main.go

FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/tinystoredb-server /usr/local/bin/tinystoredb-server
COPY --from=builder /app/tinystoredb-cli /usr/local/bin/tinystoredb-cli
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

# Create symlinks in /app and /usr/local/bin for convenience
RUN ln -s /usr/local/bin/tinystoredb-cli /usr/local/bin/tinystoredb && \
    ln -s /usr/local/bin/tinystoredb-server /app/tinystoredb-server && \
    ln -s /usr/local/bin/tinystoredb-cli /app/tinystoredb-cli && \
    chmod +x /usr/local/bin/docker-entrypoint.sh

# Create data directory
RUN mkdir -p /data
ENV TINYSTOREDB_PORT=7389
ENV TINYSTOREDB_DATA_DIR=/data

EXPOSE 7389

ENTRYPOINT ["docker-entrypoint.sh"]
