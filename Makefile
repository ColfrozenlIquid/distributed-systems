docker-compose-up:
	@docker compose up -d

docker-compose-up-build:
	@docker compose up -d --build

docker-compose-down:
	@docker compose down --remove-orphans

.PHONY: generate-proto docker-compose-up-build

generate-proto:
	@for d in services/*; do \
		if [ -f "$$d/Makefile" ]; then \
			echo "Generating proto in $$d"; \
			$(MAKE) -C "$$d" generate-proto; \
		fi \
	done

aufgabe1-functional: docker-compose-up-build
	docker run --rm \
	  --network vs-praktikum_default \
	  -v "$$PWD/tests/aufgabe1:/tests" \
	  -w /tests \
	  -e CONTROL_CENTER_URL=http://control-center:8080 \
	  golang:1.26.2-alpine \
	  go test -v -run TestRegistrationCriticalReadingAndDashboard .


aufgabe1-nonfunctional: docker-compose-up-build
	docker run --rm \
	  --network vs-praktikum_default \
	  -v "$$PWD/tests/aufgabe1:/tests" \
	  -w /tests \
	  -e CONTROL_CENTER_URL=http://control-center:8080 \
	  golang:1.26.2-alpine \
	  go test -v -run TestStatusEndpointAvailabilityAndLatency .

aufgabe2-functional: docker-compose-up-build
	docker run --rm \
		--network vs-praktikum_default \
		-v "$PWD/tests/aufgabe2:/tests" \
		-w /tests \
		-e CONTROL_CENTER_URL=http://control-center:8080 \
		golang:1.26.2-alpine \
		go test -v -run TestAssignWaterIncidentMissionViaGRPC .

aufgabe2-nonfunctional: docker-compose-up-build
	docker run --rm \
		--network vs-praktikum_default \
		-v "$PWD/tests/aufgabe2:/tests" \
		-w /tests \
		-e CONTROL_CENTER_URL=http://control-center:8080 \
		golang:1.26.2-alpine \
		go test -v -run TestMissionAssignmentStreamLatencyAndReliability .

aufgabe3-functional: docker-compose-up-build
	docker run --rm \
	  --network vs-praktikum_default \
	  -v "$PWD/tests/aufgabe3:/tests" \
	  -w /tests \
	  -e CONTROL_CENTER_URL=http://control-center:8080 \
	  -e MQTT_BROKER_URL=tcp://mqtt-broker:1883 \
	  golang:1.26.2-alpine \
	  go test -v -run TestMQTTSensorAndActorMessagesReachControlCenter .

aufgabe4-functional: docker-compose-up-build
	docker run --rm --network vs-praktikum_default \
	  -v "$PWD/tests/aufgabe4:/tests" \
	  -w /tests \
	  -e CONTROL_CENTER_URL=http://control-center:8080 \
	  -e MQTT_BROKER_URL=tcp://mqtt-broker:1883 \
	  golang:1.26.2-alpine \
	  go test -v -run TestRicartAgrawalaMutualExclusionAndLogicalClocks .