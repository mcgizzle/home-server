FROM golang:1.23.1-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED=1
RUN apk add --no-cache gcc libc-dev

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN go build -o cloud .

# Runtime image
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/cloud .

EXPOSE 8089

RUN mkdir -p data

COPY --from=builder /app/static static
COPY --from=builder /app/internal/infrastructure/migrations internal/infrastructure/migrations

CMD ["./cloud"]