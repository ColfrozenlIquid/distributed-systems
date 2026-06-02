build:
	go build -o actor ./cmd/actors/

generate-proto:
	mkdir -p gen
	protoc --proto_path=../../shared/proto \
		--go_out=gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen \
		--go-grpc_opt=paths=source_relative \
		../../shared/proto/world.proto \
		../../shared/proto/actor.proto

assign-mission:
	grpcurl -plaintext \
		-d '{"mission":{"id":{"uuid":"mission-123"},"type":"MISSION_TYPE_PUMP_WATER","position":{"x":14,"y":12},"priority":"MISSION_PRIORITY_HIGH"}}' \
		localhost:50051 \
		ActorService/AssignMission
