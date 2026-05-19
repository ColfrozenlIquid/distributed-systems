FROM golang:1.26.2-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/simulation/go.mod services/simulation/go.sum ./services/simulation/
WORKDIR /app/services/simulation
RUN go mod download

COPY services/simulation/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" -o app ./cmd/simulation/

RUN wget -qO /bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.28/grpc_health_probe-linux-amd64 \
    && chmod +x /bin/grpc_health_probe

FROM scratch
COPY --from=builder /app/services/simulation/app /app
COPY --from=builder /bin/grpc_health_probe /grpc_health_probe
ENTRYPOINT ["/app"]
