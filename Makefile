.PHONY: arch-check build lint test check-coverage test-integration test-e2e gate-all clean install-mutation-tools mutation-test mutation-test-pr

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

# Mutation testing with gomutants (https://github.com/szhekpisov/gomutants).
MUTATION_TOOL_VERSION ?= v0.3.0
GOMUTANTS ?= $(HOME)/go/bin/gomutants

install-mutation-tools:
	go install github.com/szhekpisov/gomutants@$(MUTATION_TOOL_VERSION)

mutation-test:
	$(GOMUTANTS) --config .gomutants.yml ./internal/...

mutation-test-pr:
	$(GOMUTANTS) --config .gomutants.yml --changed-since origin/main --threshold-efficacy 100 --threshold-mcover 100 ./internal/...

mutation-test-dry:
	$(GOMUTANTS) --config .gomutants.yml --dry-run ./internal/...

test-integration:
	rm -f testdata/qbt-config/lockfile
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit integration-tests
	docker compose -f docker-compose.test.yml down

gate-all: build lint test check-coverage arch-check test-integration mutation-test-pr

clean:
	rm -f coverage.out
	rm -f bot
