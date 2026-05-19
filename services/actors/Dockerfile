FROM golang:1.26.2-alpine AS builder

WORKDIR /app
COPY shared/ ./shared/
COPY services/actors/go.mod services/actors/go.sum ./services/actors/
WORKDIR /app/services/actors
RUN go mod download

COPY services/actors/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" -o actor .

FROM scratch

COPY --from=builder /app/services/actors/actor /actor

ENTRYPOINT ["/actor"]
