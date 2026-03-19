# --- Build Stage ---
# Use the platform of the build machine (BUILDPLATFORM) to run the compiler
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

# Arguments provided by Docker Buildx automatically
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Map TARGETOS/TARGETARCH to GOOS/GOARCH for a cross-platform static build
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o /nekotree ./cmd/nekotree

# --- Runtime Stage ---
FROM alpine:latest
RUN apk add --no-cache git bash docker-cli

WORKDIR /workspace

# Copy the binary specifically built for the target architecture
COPY --from=builder /nekotree /usr/local/bin/nekotree

CMD ["tail", "-f", "/dev/null"]
