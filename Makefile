.PHONY: docker-up docker-down dev-frontend dev-server dev-agent dev install

# Docker commands
docker-up:
	docker compose up --build

docker-down:
	docker compose down -v

# Local development
dev-frontend:
	cd frontend && npm run dev

dev-server:
	cd server && go run ./cmd/server

dev-agent:
	cd agents/code-agent && go run .

dev:
	make -j3 dev-frontend dev-server dev-agent

# Install dependencies
install:
	cd frontend && npm install
	cd server && go mod tidy
	cd agents && go mod tidy

# Build verification
build-check:
	cd server && go build ./cmd/server
	cd agents && go build -o /tmp/code-agent-check ./code-agent
	cd frontend && npx tsc --noEmit
