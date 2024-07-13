# Stage 1: Build the Go application
FROM golang:1.22 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Cache dependencies by copying go.mod and go.sum separately and running go mod download
COPY go.mod go.sum ./
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .

# Stage 2: Copy the binary into a lightweight image
FROM gcr.io/distroless/base-debian10

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the prebuilt binary from the builder stage
COPY --from=builder /app/main .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
