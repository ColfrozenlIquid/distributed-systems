FROM golang:1.26.2-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY vshttp/ ./vshttp/
COPY services/simulation/ ./services/simulation/
COPY services/control-center/go.mod services/control-center/go.sum ./services/control-center/
WORKDIR /app/services/control-center
RUN go mod download

COPY services/control-center/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" -o app .

FROM scratch
COPY --from=builder /app/services/control-center/app /app
ENTRYPOINT ["/app"]
