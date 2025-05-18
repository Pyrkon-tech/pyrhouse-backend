# Pyrhouse Management Service
Pyrhouse is a dedicated backend platform written in go to serve as simple warehouse management platform  

Table of contents:  
1. [Core funtionalities](#core-functionalities)  
2. [Configuring and running application](#configuring-and-running-application)  
    * Catalogue Structure  
    * How to start?  
    * How to create migration  
    * Utility commands
    * .env configuration
3. [Production configuration](#production-configuration)
    
4. [Optional knowledge](#nice-to-know)

## Core functionalities
### Base features:
- Manage Equipment
    - Assets - equipment with serial number like printer, laptop
    - Stock Items - things you want to manage using quantity like powercord or cheap cables
    - Category management to easilt
- Location management - Default warehouse and equipment possible locations
- Transfer Management - it allows you to send equipment to different location and track activity
- User Management / Access Management - on a very basic level

### Additional features:
- Build in Service Desk, application was created for IT department on a convention, mass party, so simple service desk functionallity can help with helping users/customers with their issues
- Google Sheets support, dedicated for a specific spreadsheet table format to get lsit of expected tasks/transfers

## Configuring and running application:

### Catalogue Structure
Structure initially based on GoLang official docs

### How to start?
1. Setup `.env` according to a recomendation below
2. `docker-compose up -d` (*there is only DB container, so you can always* `docker run...`)
3. `go run main.go`
4. *Test if applications works by simply calling* `[GET] {app-url}/health`

> that's all folks, check `/docs/openapi.yaml` for additional endpoints specification

### How to create migration
- `brew install golang-migrate` *(optional) on mac if never installed*
- `migrate create -ext sql -dir ./migrations -seq name-init_table`

### Useful commands
- `make migrate` - manually execute db migrations
- `go mod tidy` - after adding module/need a package  
- `go build cmd/server` + `./cmd/server/` (`server.main`) - manual application build
- `docker-compose down -v` -> remove volume to repopulate sql

### Envs to run application:
```
# Core
DATABASE_URL
APP_ROOT_PATH // just set . in that case

# Recommended, if IT Form to request purchases/order equipment will be still in Google Sheets
GOOGLE_SHEETS_CREDENTIALS_JSON // Json file to startup quest board capabilites to spreadsheet, usually provided by google in a form {"type":"service_account","project_id":"...}

# Specific, if you want to continue developing integration with jira service desk
JIRA_API_TOKEN
JIRA_BASE_URL
JIRA_EMAIL
JIRA_SERVICE_DESK_ID
JWT_SECRET

# Optional
PORT // on which port to setup app, default 8080
REQUEST_TIMEOUT
```

## Production configuration
Application infrastructure was originally setup on digital ocean for build and run we are using `./Dockerfile`  
Build runs `./start.sh` to execute migration upon container start
> There was no CI/CD pipeline build, app utilizes digital ocean hook on github push 


### Nice to know
Jira service is only initialize in container and in case of any error, it has silent kill in `internal/core/container/container.go`

```
jiraHandler, err := jira.NewJiraHandler()

if err != nil {
    jiraHandler = nil
}
```