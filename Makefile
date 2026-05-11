.PHONY: arch-check build lint test check-coverage test-integration test-e2e gate-all clean

arch-check:
	go run github.com/arch-go/arch-go/v2@latest

build:
	go build ./...

lint:
	golangci-lint run

test:
	go test ./... -short -cover -coverprofile=coverage.out

COVERAGE_THRESHOLD ?= 80
check-coverage:
	@threshold=$(COVERAGE_THRESHOLD); \
	total=$$(go tool cover -func=coverage.out 2>/dev/null | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$total" ]; then echo "FAIL: no coverage data"; exit 1; fi; \
	if [ "$$(echo "$$total < $$threshold" | bc)" -eq 1 ]; then \
		echo "FAIL: coverage $$total% < $$threshold%"; exit 1; \
	else \
		echo "PASS: coverage $$total% >= $$threshold%"; \
	fi

test-integration:
	rm -f testdata/qbt-config/lockfile
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit integration-tests
	docker compose -f docker-compose.test.yml down

gate-all: build lint test check-coverage arch-check

clean:
	rm -f coverage.out
	rm -f bot
