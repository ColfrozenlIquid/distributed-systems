generate-proto:
	mkdir -p gen
	protoc --proto_path=../shared/proto \
		--go_out=gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen \
		--go-grpc_opt=paths=source_relative \
		../shared/proto/performance.proto \

		MQTT_BROKER_HOST ?= localhost


SERVER_HOST ?= localhost
MQTT_BROKER_HOST ?= localhost
MQTT_BROKER_PORT ?= 1883

run-broker:
	docker run -it -p $(MQTT_BROKER_PORT):1883 \
	-v $(CURDIR)/mosquitto.conf:/mosquitto/config/mosquitto.conf \
	eclipse-mosquitto

run-server:
	MQTT_BROKER_HOST=$(MQTT_BROKER_HOST) MQTT_BROKER_PORT=$(MQTT_BROKER_PORT) go run ./cmd/server/

run-client:
	MQTT_BROKER_HOST=$(MQTT_BROKER_HOST) \
	MQTT_BROKER_PORT=$(MQTT_BROKER_PORT) \
	GRPC_SERVER_ADDR=$(SERVER_HOST):50051 \
	go run ./cmd/client

.PHONY: generate-proto run-broker run-server run-client

build-server:
	go build ./cmd/server
