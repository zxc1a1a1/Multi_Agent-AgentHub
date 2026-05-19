.PHONY: api web dev test fmt tidy

api:
	cd apps/api && go run ./cmd/server

web:
	cd apps/web && npm run dev

dev:
	@echo "Start API and Web in two terminals: make api / make web"

tidy:
	cd apps/api && go mod tidy

fmt:
	cd apps/api && gofmt -w .

test:
	cd apps/api && go test ./...
