# Hello Worlds

## Catalogue Structure
Structure based on GoLang official docs meh

## Useful things
`go mod tidy` - after adding module/need a package  
`go build cmd/server` + `./cmd/server/` (`server.main`)  
`docker-compose down -v` -> remove volume to repopulate sql

## How to start?
1. `docker-compose up -d`
2. `go run main.go`

yop, that's all

## Utils

#### How to create migration
- `brew install golang-migrate`  
- `migrate create -ext sql -dir ./migrations -seq name-init_table`