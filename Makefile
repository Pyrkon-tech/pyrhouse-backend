APP_PORT      ?= 8081
DB_CONTAINER  ?= go-test-db-postgres-1
DB_USER       ?= postgres
DB_PASSWORD   ?= pyrpyr
DB_PORT       ?= 15432
TEST_DB_NAME  ?= pyrhouse_test
TEST_DB_URL   ?= postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(TEST_DB_NAME)?sslmode=disable

.PHONY: start
start: ## start postgres (if not running) then launch the app
	docker-compose up -d
	go run ./main.go

.PHONY: stop
stop: ## stop the app process then bring down postgres
	-lsof -ti :$(APP_PORT) | xargs kill -SIGTERM 2>/dev/null || true
	docker-compose down

.PHONY: test
test: _test-db-setup ## run all tests including integration (against pyrhouse_test)
	TEST_DATABASE_URL="$(TEST_DB_URL)" go test ./... -count=1 -timeout=120s

.PHONY: test-ci
test-ci: _test-db-setup ## run all tests then tear down postgres (for CI)
	TEST_DATABASE_URL="$(TEST_DB_URL)" go test ./... -count=1 -timeout=120s; \
	  EXIT_CODE=$$?; \
	  docker-compose down; \
	  exit $$EXIT_CODE

.PHONY: _test-db-setup
_test-db-setup: ## (internal) ensure test DB exists and is migrated
	docker-compose up -d
	docker exec $(DB_CONTAINER) psql -U $(DB_USER) -c "CREATE DATABASE $(TEST_DB_NAME);" 2>/dev/null || true
	DATABASE_URL="$(TEST_DB_URL)" go run ./main.go -migrate -dir=./migrations

.PHONY: build
build: ## build app
	@mkdir -p build
	go build ./main.go

.PHONY: run
run: ## run app
	go run ./main.go

.PHONY: migrate
migrate: ## run migrations against dev database
	go run ./main.go -migrate -dir=./migrations

.PHONY: migrate-only
migrate-only: ## run only migrations without starting the server
	go run ./cmd/migrate/main.go --dir=./migrations

# .PHONY: fixtures
# fixtures: ## run app migrations
