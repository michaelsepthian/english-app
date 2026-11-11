FROM node:24-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build
FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /english-app .
FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && addgroup -S english && adduser -S english -G english
WORKDIR /app
COPY --from=backend /english-app ./english-app
COPY --from=frontend /src/frontend/dist ./web
ENV HTTP_ADDR=:8080 STATIC_DIR=/app/web TZ=Asia/Jakarta
USER english
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
CMD ["./english-app"]
