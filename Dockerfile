# Build stage
FROM golang:1.22.4 AS builder

# Set the working directory for the build
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire project into the container
COPY . .

# Set the working directory to where main.go is located
WORKDIR /app/cmd/server

# Build the Go app
RUN go build -o /main .

# Final stage
FROM alpine:latest

# Set working directory in container
WORKDIR /root/

# Copy the built binary from the builder
COPY --from=builder /main .

# Expose port 8080
EXPOSE 8080

# Run the binary
CMD ["./main"]