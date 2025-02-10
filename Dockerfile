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

# Create entrypoint script to setup SSH agent
RUN echo '#!/bin/sh\n\
eval $(ssh-agent -s)\n\
if [ -d "/root/.ssh" ]; then\n\
  for key in /root/.ssh/id_*; do\n\
    if [ -f "$key" ] && [ "${key%.pub}" = "$key" ]; then\n\
      ssh-add "$key"\n\
    fi\n\
  done\n\
fi\n\
exec "$@"' > /entrypoint.sh && chmod +x /entrypoint.sh

EXPOSE 3000
ENTRYPOINT ["/entrypoint.sh"]
CMD ["upgist"]
