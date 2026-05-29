.PHONY: docker-up docker-down dev-frontend dev-server dev-agent dev dev-gateway-new dev-code-agent-new dev-web-agent-new dev-new-arch install smoke-test smoke-test-ci smoke-multi-agent smoke-multi-agent-ps

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

# New architecture local demo entry points
dev-gateway-new:
	go run ./services/gateway/cmd/gateway

dev-code-agent-new:
	go run ./services/agents/code-agent/cmd/code-agent

dev-web-agent-new:
	go run ./services/agents/web-agent/cmd/web-agent

dev-new-arch:
	make -j3 dev-gateway-new dev-code-agent-new dev-web-agent-new

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

# Smoke test (keeps services running for manual demo)
smoke-test:
	bash ./smoke-test.sh --keep

# Smoke test (tears down after — for CI)
smoke-test-ci:
	bash ./smoke-test.sh --down

smoke-multi-agent:
	bash ./smoke-multi-agent-sse.sh --down

smoke-multi-agent-ps:
	powershell -ExecutionPolicy Bypass -File ./smoke-multi-agent-sse.ps1
