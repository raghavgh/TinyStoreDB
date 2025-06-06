FROM golang:1.24-alpine as builder

WORKDIR /app
COPY . .

# Build binary
RUN go build -o tinystoredb main.go

FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/tinystoredb .

# Create data directory
RUN mkdir -p /data
ENV TINYSTOREDB_PORT=50051
ENV TINYSTOREDB_DATA_DIR=/data

EXPOSE 50051

ENTRYPOINT ["./tinystoredb"]