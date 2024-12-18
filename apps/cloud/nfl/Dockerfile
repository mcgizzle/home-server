FROM golang:1.23.1-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED=1
RUN apk add --no-cache gcc libc-dev

# Copy the go modules
COPY go.mod ./
COPY go.sum ./

# Cache dependencies
RUN go mod download

COPY data/results.db .

# Copy the source code
COPY . .

# Build the application
RUN go build -o cloud .

# Runtime image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/cloud .

# Expose the port (if applicable)
EXPOSE 8089

# Create a directory for the database
RUN mkdir -p data

# Copy the database if it exists
COPY --from=builder /app/data/results.db data/results.db
COPY --from=builder /app/static static


# Command to run the application
CMD ["./cloud"]