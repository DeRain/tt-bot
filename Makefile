.PHONY: arch-check build lint test test-integration test-e2e gate-all clean

arch-check:
	go run github.com/arch-go/arch-go/v2@latest

build:
	go build ./...

lint:
	golangci-lint run

test:
	go test ./... -short -cover -coverprofile=coverage.out

test-integration:
	rm -f testdata/qbt-config/lockfile
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit integration-tests
	docker compose -f docker-compose.test.yml down

gate-all: build lint test arch-check

clean:
	rm -f coverage.out
	rm -f bot
