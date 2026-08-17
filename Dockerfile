# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.23-alpine AS go-builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=go-builder /app/server .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN mkdir -p data/local
EXPOSE 8080
CMD ["./server"]
