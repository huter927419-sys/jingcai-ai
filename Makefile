.PHONY: dev backend web tidy test build

dev:
	$(MAKE) -j2 backend web

backend:
	go run ./cmd/server

web:
	cd web && npm run dev

tidy:
	go mod tidy

test:
	go test ./...

build:
	cd web && npm run build
	go build -o bin/server ./cmd/server
