# Stage 1: Build Svelte frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY middleware/services/dashboard/frontend/package.json .
RUN npm install
COPY middleware/services/dashboard/frontend/ .
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-bookworm AS go-builder
WORKDIR /build
COPY middleware/ middleware/
WORKDIR /build/middleware/services/dashboard
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -o /dashboard .

# Stage 3: Final image
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-builder  /dashboard               /dashboard
COPY --from=frontend-builder /app/dist           /frontend/dist
EXPOSE 8080
ENTRYPOINT ["/dashboard"]
