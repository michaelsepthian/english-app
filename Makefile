.PHONY: up down test logs
up:
	BUILDX_CONFIG=/tmp/english-app-buildx docker compose up -d --build
down:
	docker compose down
test:
	./scripts/test.sh
logs:
	docker compose logs -f --tail=50

.PHONY: format format-frontend format-python format-go format-check
format: format-frontend format-python format-go
format-frontend:
	npm --prefix frontend run format
format-python:
	python3 -m black scripts
format-go:
	./scripts/format-go.sh --write
format-check:
	npm --prefix frontend run format:check
	python3 -m black --check scripts
	./scripts/format-go.sh --check
