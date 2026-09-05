# =============================================================================
# CrawlIQ Makefile
#
# Common development tasks. Run `make help` to see what's available.
# =============================================================================

GO         ?= go
APP        := crawliq
BIN_DIR    := bin
BINARY     := $(BIN_DIR)/$(APP)
MAIN_PKG   := ./cmd/api
DOCKER     := docker
COMPOSE    := docker compose

# Default DB connection used by `make migrate`. Override with `make
# migrate DB_URL=...` if your local setup differs.
DB_URL     ?= postgres://postgres:password@localhost:5432/crawliq?sslmode=disable

.PHONY: help
help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: all
all: tidy vet test build ## Run tidy + vet + test + build in one shot.

.PHONY: build
build: ## Build the API binary into ./bin/crawliq.
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags="-s -w" -o $(BINARY) $(MAIN_PKG)

.PHONY: run
run: ## Run the API locally (reads config/config.yaml).
	$(GO) run $(MAIN_PKG)

.PHONY: tidy
tidy: ## Run `go mod tidy`.
	$(GO) mod tidy

.PHONY: vet
vet: ## Run `go vet` across the whole module.
	$(GO) vet ./...

.PHONY: test
test: ## Run all unit tests.
	$(GO) test ./... -count=1

.PHONY: test-race
test-race: ## Run unit tests with the race detector.
	$(GO) test ./... -race -count=1

.PHONY: fmt
fmt: ## Run `gofmt -s -w` over the whole tree.
	$(GO) fmt ./...

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(BIN_DIR)

.PHONY: docker-build
docker-build: ## Build the Docker image (tag: crawliq:latest).
	$(DOCKER) build -t $(APP):latest .

.PHONY: docker-up
docker-up: ## Start the full local stack (Postgres + migrations + API).
	$(COMPOSE) up --build

.PHONY: docker-down
docker-down: ## Stop and remove the local stack.
	$(COMPOSE) down -v

.PHONY: migrate
migrate: ## Apply database migrations using goose.
	@which goose >/dev/null || (echo "goose not installed. Run: go install github.com/pressly/goose/v3/cmd/goose@latest" && exit 1)
	goose -dir migrations up "$(DB_URL)"