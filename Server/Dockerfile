# --- Stage 1: Build Frontend ---
FROM node:20-alpine AS frontend-builder
WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# --- Stage 2: Build Backend ---
FROM golang:1.24-alpine AS backend-builder
# CGO_ENABLED=1 is required for go-sqlite3
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Inject the built frontend assets into the backend's web directory
RUN rm -rf web/* && mkdir -p web
COPY --from=frontend-builder /build/dist ./web/
# Build the Go executable
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o minepannel-server .

# --- Stage 3: Final Runtime ---
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

# Copy the backend binary and the web directory
COPY --from=backend-builder /build/minepannel-server /app/
COPY --from=backend-builder /build/web /app/web/
COPY entrypoint.sh /app/

RUN chmod +x /app/entrypoint.sh

# Expose management and business ports
EXPOSE 55000 55001

ENTRYPOINT ["/app/entrypoint.sh"]
