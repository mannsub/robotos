FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY middleware/go.mod middleware/go.sum ./
COPY middleware/services/neodm/proto/neodmpb/go.mod ./services/neodm/proto/neodmpb/go.mod
COPY middleware/services/neodm/proto/neodmpb/go.sum ./services/neodm/proto/neodmpb/go.sum
RUN go mod download

COPY middleware/ .
RUN CGO_ENABLED=0 GOARCH=arm64 go build -o /bin/robotos ./cmd/robotos/...

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/robotos /robotos
ENTRYPOINT ["/robotos"]