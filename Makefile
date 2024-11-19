.PHONY: build
build: ## build app
	@mkdir -p build
	go build ./main.go


.PHONY: run
run: ## run app
	go run ./main.go

.PHONY: migrate
migrate: ## run app migrations
	go run ./main.go migrate --dir=./migrations
# .PHONY: fixtures
# fixtures: ## run app migrations
