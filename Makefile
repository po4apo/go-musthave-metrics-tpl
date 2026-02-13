.PHONY: run-server run-agent build test fmt vet check

run-server:
	go run ./cmd/server
run-agent:
	go run ./cmd/agent -r 2 -p 1
build:
	go build ./...

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

# Полная проверка перед вливанием: форматирование, vet, тесты, сборка
check: fmt vet test build
