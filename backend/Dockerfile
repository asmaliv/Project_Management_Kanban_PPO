# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build statically-linked binary without debug symbols
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /app/server ./cmd/server

# Stage 2: Minimal non-root runtime image
FROM alpine:3.21 AS runner

RUN apk add --no-cache ca-certificates tzdata

# Create dedicated non-root user and group
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy binary and database migrations
COPY --from=builder --chown=appuser:appgroup /app/server /app/server
COPY --chown=appuser:appgroup migrations /app/migrations

USER 10001:10001

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/server"]
