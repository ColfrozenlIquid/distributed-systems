DOCKER_TEST_IMAGE := golang:1.26.2-alpine
DOCKER_NETWORK := vs-praktikum_default
CONTROL_CENTER_URL := http://control-center:8080
MQTT_BROKER_URL := tcp://mqtt-broker:1883

define run-go-test
	docker run --rm \
	  --network $(DOCKER_NETWORK) \
	  -v "$$PWD/tests/$(1):/tests" \
	  -w /tests \
	  -e CONTROL_CENTER_URL=$(CONTROL_CENTER_URL) \
	  -e MQTT_BROKER_URL=$(MQTT_BROKER_URL) \
	  $(DOCKER_TEST_IMAGE) \
	  go test -v -run $(2) .
endef

docker-compose-up:
	@docker compose up -d

docker-compose-up-build:
	@docker compose up -d --build

docker-compose-down:
	@docker compose down --remove-orphans

.PHONY: generate-proto docker-compose-up docker-compose-up-build docker-compose-down \
	aufgabe1-functional aufgabe1-nonfunctional \
	aufgabe2-functional aufgabe2-nonfunctional \
	aufgabe3-functional aufgabe4-functional

generate-proto:
	@for d in services/*; do \
		if [ -f "$$d/Makefile" ]; then \
			echo "Generating proto in $$d"; \
			$(MAKE) -C "$$d" generate-proto; \
		fi \
	done

aufgabe1-functional: docker-compose-up-build
	@$(call run-go-test,aufgabe1,TestRegistrationCriticalReadingAndDashboard)


aufgabe1-nonfunctional: docker-compose-up-build
	@$(call run-go-test,aufgabe1,TestStatusEndpointAvailabilityAndLatency)

aufgabe2-functional: docker-compose-up-build
	@$(call run-go-test,aufgabe2,TestAssignWaterIncidentMissionViaGRPC)

aufgabe2-nonfunctional: docker-compose-up-build
	@$(call run-go-test,aufgabe2,TestMissionAssignmentStreamLatencyAndReliability)

aufgabe3-functional: docker-compose-up-build
	@$(call run-go-test,aufgabe3,TestMQTTSensorAndActorMessagesReachControlCenter)

aufgabe4-functional: docker-compose-up-build
	@$(call run-go-test,aufgabe4,TestRicartAgrawalaMutualExclusionAndLogicalClocks)