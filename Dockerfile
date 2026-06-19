# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
# We'll create main.go in the next phase, but setting up the build step now
RUN go build -o mycli .

# Run stage
FROM alpine:latest
WORKDIR /app

# Copy the built binary
COPY --from=builder /app/mycli .

# No CMD or ENTRYPOINT yet because we will run it interactively via docker run -it
ENTRYPOINT ["./mycli"]
