# Metric Generator

This Go application generates consistent metrics for a set of servers and sends them to an Elasticsearch instance.

## Prerequisites

- [Go](https://golang.org/doc/install) installed on your machine.
- [Docker](https://www.docker.com/get-started) installed on your machine.
- An Elasticsearch instance running (you can use Docker for this).

## Configuration

The application uses environment variables for configuration. You can specify these variables in a `.env` file in the root directory of your project.

### Example `.env` file:

```plaintext
SERVER_COUNT=100
ES_SERVER=http://localhost:9200
ES_USERNAME=
ES_PASSWORD=
ES_INDEX=server-metrics
```

## Docker

### Dockerfile

Here's the example `Dockerfile` for running this Go application:

```Dockerfile
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
COPY --from=builder /app/.env /.

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["/main"]
```

### Building the Docker Image

To build the Docker image for the Go application, run the following commands in the root directory of the repository:

```sh
docker build -t metric-generator .
```

### Running the Docker Container

After building the Docker image, you can run the container using the following command:

```sh
docker run --env-file .env metric-generator
```

### Elasticsearch

You can run an Elasticsearch instance using Docker if you don't already have one. Here is the command to start Elasticsearch:

```sh
docker run -d -p 9200:9200 -e "discovery.type=single-node" docker.elastic.co/elasticsearch/elasticsearch:8.4.0
```

Ensure that the `ES_SERVER` in your `.env` file points to this instance, e.g., `ES_SERVER=http://localhost:9200`.

## Running Locally

If you want to run the application directly without Docker, follow these steps:

1. Ensure you have the `.env` file configured in the root directory.
2. Build the Go application:

    ```sh
    go build -o main .
    ```

3. Run the application:

    ```sh
    ./main
    ```

## Contributing

Feel free to open issues or submit pull requests if you have any improvements or bug fixes.
