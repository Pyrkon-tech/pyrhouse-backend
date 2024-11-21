# Stage 1: Build the static HTML for OpenAPI using Redoc CLI
FROM node:18-alpine AS redoc-builder

RUN npm install -g redoc-cli
WORKDIR /docs
COPY docs/openapi.yaml .
RUN redoc-cli bundle openapi.yaml -o index.html

# Stage 2: Build the Go application
FROM golang:1.22.4 AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o main .

# Stage 3: Final image with static HTML and Go application
FROM alpine:3.18

# Set working directory
WORKDIR /app

# Copy the Go application binary from the builder stage
COPY --from=go-builder /app/main .

# Copy the generated static HTML file from the Redoc stage
COPY --from=redoc-builder /docs/index.html ./static/index.html

# Expose port 8080 for the Go application
EXPOSE 8080

# Run the application
CMD ["./main"]