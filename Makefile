.PHONY: build lint test test-integration test-e2e gate-all clean

build:
	go build ./...

lint:
	golangci-lint run

test:
	go test ./... -short -cover -coverprofile=coverage.out

test-integration:
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit integration-tests
	docker compose -f docker-compose.test.yml down

gate-all: build lint test

clean:
	rm -f coverage.out
	rm -f bot
