
GOLANG := golang:1.25.3
TOOLING_PATH = ./app/tooling

NAMESPACE := envoker
VERSION := 0.0.1

NAME=envoker
APP=app
APP_IMAGE := $(NAME)/$(APP):$(VERSION)
APP_IMAGE_LATEST := $(NAME)/$(APP):latest

# ---------------- BUILD
build: build-envoker

build-envoker:
	docker build \
    -f workshop/docker/dockerfile.envoker\
    -t $(APP_IMAGE) \
    -t $(APP_IMAGE_LATEST) \
    --build-arg BUILD_REF=$(VERSION) \
    --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    .


########################################
## MANAGE
######################
tidy:
	go mod tidy
	go mod vendor

# Drop and recreate database schema
db-reset-local:
	@echo "🗑️  Dropping and recreating public schema..."
	@docker exec $(NAME)-postgres psql -U db_user -d database -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

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
	@echo "  make db-reset-local                - Drop and recreate database schema"
	@echo "  make migrate                       - Run database migrations"
	@echo "  make db-reflect                    - Reflect database schema to JSON"
	@echo "  make generate-all                  - Generate code for ALL tables from JSON"
	@echo "  make generate-table TABLE=<name>   - Generate code for SINGLE table from JSON"
	@echo "  make db-code-full-reset            - Full workflow: reset -> migrate -> reflect -> generate"
	@echo ""
	@echo "Examples:"
	@echo "  make db-code-full-reset            # Complete reset and regeneration"
	@echo "  make generate-table TABLE=tasks    # Regenerate single table"
	@echo "  make generate-all                  # Regenerate all tables"
	@echo "  make db-reflect                    # Just update JSON from current DB"




	

########################################
## DEV
######################
# watch:
# 	wgo run app/api/main.go
# dev:
# 	wgo run app/envoker/main.go
	
# dev-admin:
# 	cd app/envoker/admin/react && pnpm dev
########################################
## DATA
######################
dev-data-up:
	docker-compose -p envoker -f workshop/dev/envoker-local-compose.yml up  -d

dev-data-down:
	docker-compose -p envoker -f workshop/dev/envoker-local-compose.yml down

dev-psql:
	PGPASSWORD=admin psql -h localhost -p 5432 -U postgres -d envoker

.PHONY: migrate
migrate: ## Run database migrations
	@echo "Running database migrations..."
	@go run $(TOOLING_PATH)/main.go migrate



########################################
## DOCKER RUN
######################
docker-run:
	docker run -p 3000:3000 --env-file .env $(APP_IMAGE_LATEST) 

docker-stop:
	docker stop $(APP_IMAGE_LATEST)

