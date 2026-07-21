# ---- Build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependency download secara terpisah dari source code.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Build binary statis (CGO off) agar bisa jalan di image minimal.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /halogold-api ./cmd/api

# ---- Run stage ----
FROM alpine:3.20

# ca-certificates untuk koneksi TLS (mis. ke DB terkelola).
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 appuser

COPY --from=builder /halogold-api /usr/local/bin/halogold-api

USER appuser
EXPOSE 8080

ENTRYPOINT ["halogold-api"]
