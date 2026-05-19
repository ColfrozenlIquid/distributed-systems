FROM golang:1.26.2-alpine AS builder

WORKDIR /app
COPY shared/ ./shared/
COPY services/sensors/go.mod services/sensors/go.sum ./services/sensors/
WORKDIR /app/services/sensors
RUN go mod download

COPY services/sensors/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" -o sensor ./cmd/sensors/

FROM scratch

COPY --from=builder /app/services/sensors/sensor /sensor

ENTRYPOINT ["/sensor"]
