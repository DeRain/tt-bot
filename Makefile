.PHONY: build lint test test-integration test-e2e gate-all clean

build:
	go build ./...

lint:
	golangci-lint run

test:
	go test ./... -short -cover -coverprofile=coverage.out

test-integration:
	docker compose -f docker-compose.test.yml up -d qbittorrent
	./scripts/wait-for-qbt.sh
	go test ./... -tags=integration -run Integration -v
	docker compose -f docker-compose.test.yml down

test-e2e:
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker compose -f docker-compose.test.yml down

gate-all: build lint test

clean:
	rm -f coverage.out
	rm -f bot
