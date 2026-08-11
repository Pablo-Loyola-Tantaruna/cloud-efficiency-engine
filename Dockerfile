FROM golang:1.25.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o cloud-efficiency-engine \
    ./cmd/api

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/cloud-efficiency-engine .

EXPOSE 8080

ENTRYPOINT ["./cloud-efficiency-engine"]