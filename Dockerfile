# Base image with Go installed
FROM golang:1.22.4

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire project into the container
COPY . .
COPY .env ./.env

# Set working directory to where main.go is located
WORKDIR /app/cmd/server

# Build the application
RUN go build -o main .

# Expose port 8080 for the application
EXPOSE 8080

# Run the application
CMD ["./main"]
