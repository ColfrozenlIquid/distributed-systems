generate-proto:
	mkdir -p gen
	protoc --proto_path=../../shared/proto \
		--go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		../../shared/proto/world.proto
