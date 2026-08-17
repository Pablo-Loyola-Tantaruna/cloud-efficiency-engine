FROM golang:1.26.6-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o cloud-efficiency-engine \
    ./cmd/api

FROM alpine:3.22

RUN addgroup -S -g 10001 appgroup \
    && adduser -S -D -H -u 10001 -G appgroup appuser

WORKDIR /app

COPY --from=builder /app/cloud-efficiency-engine .

RUN chown appuser:appgroup /app/cloud-efficiency-engine

USER appuser:appgroup

EXPOSE 8080

ENTRYPOINT ["./cloud-efficiency-engine"]
