ARG BUILD_TARGET=node
FROM golang:1.26.2-alpine AS builder
ARG BUILD_TARGET
WORKDIR /src
COPY shared/ ./shared/
COPY performance/ricart-agrawala/ ./performance/ricart-agrawala/
WORKDIR /src/performance/ricart-agrawala
RUN go build -o /out/app ./${BUILD_TARGET}

FROM alpine:3.20
COPY --from=builder /out/app /app
ENTRYPOINT ["/app"]
