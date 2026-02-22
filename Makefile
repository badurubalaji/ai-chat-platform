.PHONY: build-backend build-lib build-demo migrate-up dev setup test

build-backend:
	@echo "Building Backend binary..."
	cd backend && go build -o ../bin/server cmd/server/main.go

build-lib:
	@echo "Building Angular library..."
	cd frontend && npm run build mdp-ai-chat

build-demo:
	@echo "Building Demo App..."
	cd frontend && npm run build demo-app

migrate-up:
	@echo "Running migrations up..."
	./bin/migrate up

setup:
	@echo "Setting up project..."
	cd backend && go mod download
	cd frontend && npm install
	cd backend && go build -o ../bin/migrate cmd/migrate/main.go
	$(MAKE) build-backend

test:
	@echo "Running backend tests..."
	cd backend && go test ./...
	@echo "Running frontend tests..."
	cd frontend && npx ng test mdp-ai-chat --watch=false || true

dev:
	@echo "Starting dev environment..."
	docker-compose up -d postgres
	make migrate-up
	make build-backend
	./bin/server
