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
