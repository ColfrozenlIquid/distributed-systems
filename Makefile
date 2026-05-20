docker-compose-up:
	@docker compose up -d

docker-compose-up-build:
	@docker compose up -d --build

docker-compose-down:
	@docker compose down --remove-orphans