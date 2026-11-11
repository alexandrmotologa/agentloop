# Stage 1: Build React 19 Frontend Cockpit
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci || npm install
COPY web/ ./
RUN npm run build

# Stage 2: Build Single-Binary Go Application
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy compiled web assets into web/dist for go:embed
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o agentloop ./cmd/agentloop

# Stage 3: Minimal Production Container
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=go-builder /app/agentloop /usr/local/bin/agentloop
COPY examples/ /app/examples/

EXPOSE 8080
VOLUME ["/app/data"]

ENTRYPOINT ["agentloop"]
CMD ["serve", "--port", "8080", "--db", "/app/data/agentloop.db"]
