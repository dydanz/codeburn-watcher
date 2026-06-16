# Build stage
FROM golang:latest AS builder
WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/token-monitor ./cmd/token-monitor/...

# Test stage (run all tests)
FROM builder AS tester
RUN go test -v -short ./... 2>&1

# Runtime stage
FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bin/token-monitor /usr/local/bin/token-monitor
ENTRYPOINT ["token-monitor"]
CMD ["--help"]
