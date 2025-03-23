# Step 1: Build Stage - Use the official Golang image to build the app
FROM golang:1.24 AS builder

# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    DB_PATH=/app/db/mercari.sqlite3 \
    PORT=8080

# Set the working directory inside the container
WORKDIR /app

# Copy Go modules and install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project into the container
COPY . .

# Build the Go binary
RUN go build -o main .

# Step 2: Runtime Stage - Use a minimal and secure base image
FROM gcr.io/distroless/base-debian12

# Set the working directory for the runtime container
WORKDIR /app

# Copy the built binary and necessary directories
COPY --from=builder /app/main .
COPY --from=builder /app/db ./db
COPY --from=builder /app/images ./images

# Expose port 8080
EXPOSE 8080

# Command to run the server
CMD ["./main"]
