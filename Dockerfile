# Build stage
FROM golang:1.22.4 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire source code
COPY . .

# Set the working directory to the location of your main.go
WORKDIR /app/cmd/server

# Build the binary
RUN go build -o /main .

# Final stage
FROM alpine:latest

# Set working directory in the final image
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /main .

# Expose port 8080
EXPOSE 8080

# Command to run the application
CMD ["./main"]
