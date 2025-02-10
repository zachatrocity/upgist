FROM golang:1.23-alpine AS builder

# Install git for go mod download
RUN apk add --no-cache git

WORKDIR /app

# Copy go module files
COPY go.mod .

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o upgist

FROM alpine:latest

# Install git and ssh client for gist operations
RUN apk add --no-cache git openssh-client

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/upgist /usr/local/bin/
COPY static /app/static

EXPOSE 3000
CMD ["upgist"]
