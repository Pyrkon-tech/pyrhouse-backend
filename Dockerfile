# Build the Go application
FROM golang:1.23.8 AS go

# Set the working directory inside the container
WORKDIR /app

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

# Make the start script executable
RUN chmod +x start.sh

# Expose port 8080 for the application
EXPOSE 8080

# Run the application using the start script
CMD ["./start.sh"]