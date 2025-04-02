######################
# 🏗️ Build + Optional Testing Stage
######################
FROM golang:1.23-alpine AS builder

ARG TEST=false
ARG FUZZ=false
ARG FUZZTIME=10s

RUN apk add --no-cache git ca-certificates bash

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Optional test run
RUN if [ "$TEST" = "true" ]; then \
      echo "🧪 Running unit tests..." && go test -v ./...; \
    fi

# Optional fuzz run
RUN if [ "$FUZZ" = "true" ]; then \
      echo "🧬 Running fuzz tests..." && go test -fuzz=Fuzz -fuzztime=$FUZZTIME; \
    fi

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o flux-helpers .

######################
# 🏃 Runtime Stage
######################
FROM alpine:3.21

RUN apk add --no-cache ca-certificates bash

WORKDIR /workdir

COPY --from=builder /app/flux-helpers /usr/local/bin/flux-helpers

ENTRYPOINT ["flux-helpers"]

