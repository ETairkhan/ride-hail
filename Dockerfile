# Use an official Golang runtime as a parent image
FROM golang:1.24-alpine

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod tidy

# Copy the entire content of the project to /app inside the container
COPY . .

# Build the Go app
RUN go build -o main cmd/main.go

# Expose the port the app runs on
EXPOSE 8080

# Run the Go app
CMD ["./main"]
