docker-compose-up:
	@docker compose up -d

docker-compose-up-build:
	@docker compose up -d --build

docker-compose-down:
	@docker compose down --remove-orphans

.PHONY: generate-proto

generate-proto:
	@for d in services/*; do \
		if [ -f "$$d/Makefile" ]; then \
			echo "Generating proto in $$d"; \
			$(MAKE) -C "$$d" generate-proto; \
		fi \
	done

aufgabe1-functional:
	docker compose up -d --build
	docker run --rm \
	  --network vs-praktikum_default \
	  -v "$PWD/tests/aufgabe1:/tests" \
	  -w /tests \
	  -e CONTROL_CENTER_URL=http://control-center:8080 \
	  golang:1.26.2-alpine \
	  go test -v -run TestRegistrationCriticalReadingAndDashboard .


aufgabe1-nonfunctional:
	docker compose up -d --build
	docker run --rm \
	  --network vs-praktikum_default \
	  -v "$PWD/tests/aufgabe1:/tests" \
	  -w /tests \
	  -e CONTROL_CENTER_URL=http://control-center:8080 \
	  golang:1.26.2-alpine \
	  go test -v -run TestStatusEndpointAvailabilityAndLatency .
