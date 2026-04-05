# Stage 1: Dependency management
FROM golang:1.25-alpine AS deps
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# Stage 2: Documentation generator
FROM deps AS docs-gen
RUN go install github.com/swaggo/swag/cmd/swag@latest
COPY . .
RUN swag init -g main.go

# Stage 3: Builder
FROM docs-gen AS builder
# Use CGO_ENABLED=0 for a truly static binary if possible
# (Keeping it enabled if required by dependencies, but disabling it is safer for minimal images)
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o omnigram main.go

# Stage 4: Runtime
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user for security
RUN adduser -D -u 10001 appuser
USER appuser

WORKDIR /app

# Copy the binary and generated docs
COPY --from=builder /app/omnigram .
# Documentation is typically compiled into the binary via the 'docs' package import, 
# but copying the swagger files can be useful for static serving if needed.
# COPY --from=builder /app/docs ./docs

EXPOSE 8080

ENTRYPOINT ["./omnigram"]
