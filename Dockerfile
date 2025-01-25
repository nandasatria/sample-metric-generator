# First stage: build the Go application
FROM golang:1.23-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Second stage: create a small image with the Go binary
FROM scratch

# Copy the executable from the builder stage
COPY --from=builder /app/main /main

COPY .env .env 
# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["/main"]