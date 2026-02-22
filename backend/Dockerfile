# ─── Build Stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

# Install build tools (needed for some C dependencies via CGO - gofpdf is pure Go, no CGO needed)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Download dependencies first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o tripmind-server .

# ─── Runtime Stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/tripmind-server .

# Railway injects PORT at runtime
EXPOSE 8080

# Run as non-root for security
RUN addgroup -S tripmind && adduser -S tripmind -G tripmind
USER tripmind

CMD ["./tripmind-server"]
