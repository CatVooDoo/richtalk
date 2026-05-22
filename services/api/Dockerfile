# Stage 1: build
FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.mod go.sum* ./
# -mod=mod allows updating go.sum on first build when go.sum doesn't exist yet
ENV GOFLAGS="-mod=mod"
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./main.go

# Stage 2: minimal runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /

COPY --from=builder /app/server /server
COPY --from=builder /build/migrations /migrations

ENV MIGRATIONS_PATH=/migrations

EXPOSE 8080

ENTRYPOINT ["/server"]
