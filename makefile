
GOLANG := golang:1.25.3
TOOLING_PATH = ./app/tooling

NAMESPACE := envoker
VERSION := 0.0.1
PG_DEFAULT=postgres
NAME=envoker
APP=app
APP_IMAGE := $(NAME)/$(APP):$(VERSION)
APP_IMAGE_LATEST := $(NAME)/$(APP):latest

########################################
## MANAGE
######################
tidy:
	go mod tidy
	go mod vendor

# Drop and recreate database schema
db-reset-local:
	@echo "🗑️  Dropping and recreating public schema..."
	@docker exec $(NAME)-postgres psql -U postgres -d $(PG_DEFAULT) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Reflect database schema to JSON
db-reflect:
	@echo "🔍 Reflecting database schema..."
	@go run app/tooling/main.go reflect-schema -schema=public

# Generate code from reflected JSON - ALL tables
generate-all:
	@echo "🚀 Generating code for all tables from reflected schema..."
	@go run app/generators/main.go generate -json=schema/reflector/output/public.json -all -force

# Generate code from reflected JSON - SINGLE table
# Usage: make generate-table TABLE=api_keys
generate-table:
	@if [ -z "$(TABLE)" ]; then \
		echo "❌ Error: TABLE parameter is required"; \
		echo "Usage: make generate-table TABLE=api_keys"; \
		exit 1; \
	fi
	@echo "🚀 Generating code for table $(TABLE)..."
	@go run app/generators/main.go generate -json=schema/reflector/output/public.json -table=$(TABLE) -force

# Common workflow: migrate -> reflect -> generate single table
# Usage: make regen TABLE=api_keys
regen:
	@if [ -z "$(TABLE)" ]; then \
		echo "❌ Error: TABLE parameter is required"; \
		echo "Usage: make regen TABLE=api_keys"; \
		exit 1; \
	fi
	@echo "🔄 Running migrate -> reflect -> generate for table $(TABLE)..."
	@$(MAKE) migrate
	@$(MAKE) db-reflect
	@$(MAKE) generate-table TABLE=$(TABLE)
	@echo "✅ Regeneration complete!"

# Common workflow: migrate -> reflect -> generate all tables
regen-all:
	@echo "🔄 Running migrate -> reflect -> generate all..."
	@$(MAKE) migrate
	@$(MAKE) db-reflect
	@$(MAKE) generate-all
	@echo "✅ Full regeneration complete!"

# Full workflow: reset DB -> migrate -> reflect -> generate all
db-code-full-reset:
	@echo "🔄 Running full database reset and code generation..."
	@$(MAKE) db-reset-local
	@$(MAKE) migrate
	@$(MAKE) db-reflect
	@$(MAKE) generate-all
	@echo "✅ Full reset complete!"

# Generate all code - uses JSON-based generation from reflected schema
generate: generate-all

# Help for generator commands
generate-help:
	@echo "Generator Makefile Targets:"
	@echo ""
	@echo "📖 Code Generation Workflow (JSON-based from reflected schema):"
	@echo ""
	@echo "🚀 Quick Commands (most commonly used):"
	@echo "  make regen TABLE=<name>            - Migrate -> Reflect -> Generate single table"
	@echo "  make regen-all                     - Migrate -> Reflect -> Generate all tables"
	@echo "  make db-code-full-reset            - Full reset: drop DB -> migrate -> reflect -> generate all"
	@echo ""
	@echo "🔧 Individual Steps:"
	@echo "  make migrate                       - Run database migrations"
	@echo "  make db-reflect                    - Reflect database schema to JSON"
	@echo "  make generate-all                  - Generate code for ALL tables from JSON"
	@echo "  make generate-table TABLE=<name>   - Generate code for SINGLE table from JSON"
	@echo "  make db-reset-local                - Drop and recreate database schema"
	@echo ""
	@echo "Examples:"
	@echo "  make regen TABLE=api_keys          # After migration, regenerate api_keys"
	@echo "  make regen-all                     # After migration, regenerate everything"
	@echo "  make db-code-full-reset            # Nuclear option - complete reset"
	@echo "  make generate-table TABLE=tasks    # Regenerate just tasks (skip migrate/reflect)"
	@echo "  make db-reflect                    # Just update JSON from current DB"


########################################
## BUILD
######################
build: build-api 

build-api:
	docker build \
    -f workshop/docker/dockerfile.$(NAME)\
    -t $(API_IMAGE_NAME) \
    -t $(API_IMAGE_LATEST) \
    --build-arg BUILD_REF=$(VERSION) \
    --build-arg BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
    .

########################################
## DEV
######################
watch:
	wgo run app/$(NAME)/main.go

# ==============================================================================
# DATA
dev-data-up:
	docker-compose -p $(NAME) -f workshop/dev/$(NAME)-data-compose.yml up  -d

dev-data-down:
	docker-compose -p $(NAME) -f workshop/dev/$(NAME)-data-compose.yml down

dev-psql:
	PGPASSWORD=admin psql -h localhost -p 5432 -U postgres -d $(PG_DEFAULT)


.PHONY: migrate
migrate: ## Run database migrations
	@echo "Running database migrations..."
	@go run $(TOOLING_PATH)/main.go migrate


########################################
## RUN
######################
run-api:
	docker run -p 3000:3000 \
		--env-file .env \
		$(API_IMAGE_LATEST)

########################################
## TESTING
######################
# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-v:
	go test -v ./...

# Run tests and generate coverage report
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race condition detection
test-race:
	go test -race ./...

# Run a specific test (usage: make test-one PKG=./path/to/package TEST=TestName)
test-one:
	go test -v $(PKG) -run $(TEST)

# Run tests with a longer timeout for containers
test-long:
	go test -v -timeout 5m ./...
