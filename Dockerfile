# --- Stage 1: Builder ---
# This stage uses a full Go environment to build our application binaries.
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files first to leverage Docker's layer caching.
# Dependencies are downloaded only when these files change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the API server binary.
# CGO_ENABLED=0 is important for creating a static binary.
# -ldflags="-w -s" strips debug information, making the binary smaller.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /api ./cmd/api/main.go

# Build the Worker binary with the same optimizations.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /worker ./cmd/worker/main.go


# --- Stage 2: Final Image ---
# This stage uses a minimal Alpine Linux image for a small and secure footprint.
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the non-sensitive configuration file from the host.
# Secrets will be injected via .env file in docker-compose.
COPY ./configs/config.yaml ./configs/config.yaml

# Copy only the compiled binaries from the 'builder' stage.
COPY --from=builder /api .
COPY --from=builder /worker .

# This image will be used for both 'api' and 'worker' services in docker-compose.
# The actual command to run ('./api' or './worker') will be specified there.

