# Build stage
FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -tags "fts5" -o /nex .

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -r -m -s /bin/false nex

WORKDIR /app
COPY --from=builder /nex /app/nex
COPY templates/ /app/templates/
COPY static/ /app/static/

RUN mkdir -p /data && chown nex:nex /data

USER nex

ENV DB_PATH=/data/nex.db
ENV PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -sf http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/nex"]
