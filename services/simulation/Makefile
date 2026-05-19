build:
	go build -o simulation ./cmd/simulation/

generate-proto:
	mkdir -p gen
	protoc --proto_path=proto \
		--go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		proto/world.proto


TileX ?= 1
TileY ?= 1

get-tile:
	grpcurl -v -plaintext -emit-defaults
	-d '{"x_pos": $(TileX), "y_pos": $(TileY), "sensor_type": 2}'
	localhost:50051 WorldService/TileReading
