.PHONY: build
build: ## build app
	@mkdir -p build
	go build cmd/server/main.go


.PHONY: run
run: ## run app
	go run cmd/server/main.go

.PHONY: migrate
migrate: ## run app migrations
	go run cmd/server/main.go migrate --dir=./migrations
# .PHONY: fixtures
# fixtures: ## run app migrations
	
