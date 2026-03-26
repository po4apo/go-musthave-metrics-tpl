.PHONY: run-server run-agent build test fmt vet check

run-server:
	go run ./cmd/server -key 1234
run-agent:
	go run ./cmd/agent -r 2 -p 1 -key 12345
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
