# syntax=docker/dockerfile:1

######################
# 🏗️ Build Stage (Go 1.23)
######################
FROM golang:1.23-alpine AS builder

# Install CA certs and git
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy and download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the actual source code
COPY . .

# Build the statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o flux-helpers .

######################
# 🏃 Runtime Stage
######################
FROM alpine:3.18

LABEL maintainer="pat-nel87"
LABEL org.opencontainers.image.source="https://github.com/pat-nel87/flux-helpers"
LABEL org.opencontainers.image.description="CLI tool for updating FluxCD HelmRelease image tags"

RUN apk add --no-cache ca-certificates bash

WORKDIR /workdir

# Copy in the built binary
COPY --from=builder /app/flux-helpers /usr/local/bin/flux-helpers

ENTRYPOINT ["flux-helpers"]
