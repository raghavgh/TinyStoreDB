FROM golang:1.24-alpine as builder

WORKDIR /app
COPY . .

# Build server and CLI binaries
RUN go build -o tinystoredb-server main.go
RUN go build -o tinystoredb ./cmd/tinystoredb/main.go

FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/tinystoredb-server /usr/local/bin/tinystoredb-server
COPY --from=builder /app/tinystoredb /usr/local/bin/tinystoredb

# Create symlinks in /app for backwards compatibility
RUN ln -s /usr/local/bin/tinystoredb-server /app/tinystoredb-server && \
    ln -s /usr/local/bin/tinystoredb /app/tinystoredb

# Create data directory
RUN mkdir -p /data
ENV TINYSTOREDB_PORT=7389
ENV TINYSTOREDB_DATA_DIR=/data

EXPOSE 7389

ENTRYPOINT ["tinystoredb-server"]
