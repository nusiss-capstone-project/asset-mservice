# Use the official Go image with version 1.25.10
FROM golang:1.25.10-alpine AS builder 

RUN apk update && \
    apk add --no-cache \
    curl \
    "zlib>=1.3.2-r0"
    
# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod tidy

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux \
    go build -a -ldflags="-w -s" -o main main.go && \
    go clean -cache -modcache

# Create a non-root user and set permissions
RUN adduser -D -h /home/appuser appuser && \
    mkdir -p logs && \
    chown -R appuser:appuser config/ && \
    chown -R appuser:appuser logs/ && \
    chmod +r config/* && \
    chmod +w logs && \
    chmod +x main

# EXPOSE 8080

USER appuser

# Command to run the application
CMD ["./main"]