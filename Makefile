.PHONY: dev-db api web test lint build check

dev-db:
	docker compose up -d postgres

api:
	go run ./cmd/netscope

web:
	pnpm --dir web dev

test:
	go test ./...
	pnpm --dir web test

lint:
	go vet ./...
	pnpm --dir web lint
	pnpm --dir web format:check

build:
	go build ./cmd/netscope
	pnpm --dir web build

check: lint test build

