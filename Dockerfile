# Stage 1: Builder
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

# Scraper build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /scraper cmd/scraper/main.go

# Playwright build install
RUN go install github.com/playwright-community/playwright-go/cmd/playwright@latest

# Stage 2: Image

# Plawright runtime
FROM mcr.microsoft.com/playwright:v1.41.0-jammy

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy scraper
COPY --from=builder /scraper .

# Copy playwright
COPY --from=builder /go/bin/playwright /usr/local/bin/playwright

RUN playwright install --with-deps chromium

# Create directory
RUN mkdir -p /app/exports

EXPOSE 2114

ENTRYPOINT ["./scraper"]