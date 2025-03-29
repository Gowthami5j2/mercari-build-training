# Step 1: Use the official Golang image to build the app
FROM golang:1.21 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project into the container
COPY . .

# Build the Go binary
RUN go build -o main .

# Step 2: Use a lightweight image to run the binary
FROM gcr.io/distroless/base-debian12

# Set the working directory for the runtime container
WORKDIR /app

# Copy the built binary and necessary files
COPY --from=builder /app/main .
COPY --from=builder /app/db ./db
COPY --from=builder /app/images ./images

# Expose port 8080
EXPOSE 8080

# Run the binary
CMD ["./main"]
