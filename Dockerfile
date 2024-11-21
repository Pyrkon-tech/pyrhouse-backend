# Stage 1: Build the static HTML for OpenAPI using Redoc CLI
FROM node:18-alpine AS redoc-builder

RUN npm install -g redoc-cli
WORKDIR /docs
COPY docs/openapi.yaml .
RUN redoc-cli bundle openapi.yaml -o index.html

# Stage 2: Build the Go application
FROM golang:1.22.4 AS go

# Set the working directory inside the container
WORKDIR /app
# Copy docs
COPY --from=redoc-builder /docs/index.html ./docs/index.html

# Copy go.mod and go.sum files first
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download
# Copy the entire project into the container
COPY . .

# Set working directory to where main.go is located
# WORKDIR /app/cmd/server
# Build the application
RUN go build -o main .

# Expose port 8080 for the application
EXPOSE 8080

# Run the application
CMD ["./main"]