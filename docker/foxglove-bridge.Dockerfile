FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Copy module manifests first to exploit Docker layer caching.
# The replace directive in the service go.mod points to ../../,
# so the middleware module root must be present at build time.
COPY middleware/go.mod middleware/go.sum ./middleware/
COPY middleware/services/foxglove-bridge/go.mod \
     middleware/services/foxglove-bridge/go.sum \
     ./middleware/services/foxglove-bridge/

WORKDIR /app/middleware/services/foxglove-bridge
RUN go mod download

# Copy all middleware source (proto stubs + service code).
WORKDIR /app
COPY middleware/ ./middleware/

WORKDIR /app/middleware/services/foxglove-bridge
RUN CGO_ENABLED=0 GOARCH=arm64 go build -o /bin/foxglove-bridge .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/foxglove-bridge /foxglove-bridge
EXPOSE 8765
ENTRYPOINT ["/foxglove-bridge"]
