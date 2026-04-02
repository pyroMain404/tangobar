# Stage 1: Build
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@v0.2.778

WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate templ templates
RUN templ generate

# Build the application
RUN CGO_ENABLED=0 go build -o gestionale .

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/gestionale ./

# Copy assets folder from builder
COPY --from=builder /build/assets ./assets

# Create data volume directory
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./gestionale"]
