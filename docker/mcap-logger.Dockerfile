FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Copy module manifests first to exploit Docker layer caching.
# The replace directive in the service go.mod points to ../../,
# so the middleware module root must be present at build time.
COPY middleware/go.mod middleware/go.sum ./middleware/
COPY middleware/services/mcap-logger/go.mod \
     middleware/services/mcap-logger/go.sum \
     ./middleware/services/mcap-logger/

WORKDIR /app/middleware/services/mcap-logger
RUN go mod download

# Copy all middleware source (proto stubs + service code).
WORKDIR /app
COPY middleware/ ./middleware/

WORKDIR /app/middleware/services/mcap-logger
RUN CGO_ENABLED=0 GOARCH=arm64 go build -o /bin/mcap-logger .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/mcap-logger /mcap-logger
ENTRYPOINT ["/mcap-logger"]
