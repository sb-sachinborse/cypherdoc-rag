.PHONY: setup up down logs test smoke clean

setup:   ## create .env with a random DB password
	./scripts/setup.sh

up: setup   ## build and start everything
	docker compose up --build -d
	@echo "Open http://localhost:$${APP_PORT:-8080} (first start downloads the AI models, a few minutes)"

down:
	docker compose down

logs:
	docker compose logs -f backend ollama-pull

test:   ## go vet + unit tests, run inside Docker
	docker build --target test ./backend

smoke:  ## end-to-end checks against the running stack
	./scripts/smoke.sh

clean:  ## stop and DELETE all data (database and models)
	docker compose down -v
