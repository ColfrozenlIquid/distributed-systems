build:
	go build -o actor ./cmd/actors/

run:
	go run ./cmd/actors

generate-proto:
	mkdir -p gen
	protoc --proto_path=../../shared/proto \
		--go_out=gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen \
		--go-grpc_opt=paths=source_relative \
		../../shared/proto/world.proto \
		../../shared/proto/actor.proto

assign-mission1:
	grpcurl -plaintext \
		-d '{"mission":{"id":{"uuid":"mission-123"},"type":"MISSION_TYPE_PUMP_WATER","position":{"X":14,"Y":12},"priority":"MISSION_PRIORITY_HIGH"}}' \
		localhost:50052 \
		ActorService/AssignMission

assign-mission2:
	grpcurl -plaintext \
		-d '{"mission":{"id":{"uuid":"mission-123"},"type":"MISSION_TYPE_PUMP_WATER","position":{"X":10,"Y":2},"priority":"MISSION_PRIORITY_HIGH"}}' \
		localhost:50052 \
		ActorService/AssignMission
