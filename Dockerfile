# Use an official Golang runtime as the base image
FROM golang:1.16

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Install any necessary dependencies
RUN go mod download

# Build the Go application
RUN go build -o main .

# Set the entry point command for the container
CMD ["./main"]