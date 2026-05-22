.PHONY: docker-up docker-down dev-frontend dev-server dev-agent dev

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v

dev-frontend:
	cd frontend && npm run dev

dev-server:
	cd server && go run ./cmd/server

dev-agent:
	cd agents/code-agent && go run .

dev:
	make -j3 dev-frontend dev-server dev-agent
