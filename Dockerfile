# Dockerfile
FROM golang:1.21-alpine

WORKDIR /app

# Install PostgreSQL client
RUN apk add --no-cache postgresql-client

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the application
RUN go build -o main .

# Expose port
EXPOSE 8085

# Run the application
CMD ["./main"]


