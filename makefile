
GOLANG := golang:1.25.1

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

# Generate all pgx stores (legacy)
generate-pgxstores:
	@echo "Generating pgx stores..."
	@find ./core/repositories -type f -name "store.go" -path "*/stores/*pgxstore/*" -execdir go generate \;

# Generate from specific SQL file (all layers)
# Usage: make generate-sql SQL=schema/users.sql
generate-resource:
	@if [ -z "$(SQL)" ]; then \
		echo "❌ Error: SQL parameter is required"; \
		echo "Usage: make generate-sql SQL=schema/users.sql"; \
		exit 1; \
	fi
	@echo "🚀 Generating all layers from $(SQL)..."
	@go run app/generators/main.go generate -sql=$(SQL) -force

# Generate only Repository layer
# Usage: make generate-repository SQL=schema/users.sql
generate-repository:
	@if [ -z "$(SQL)" ]; then \
		echo "❌ Error: SQL parameter is required"; \
		echo "Usage: make generate-repository SQL=schema/users.sql"; \
		exit 1; \
	fi
	@echo "📦 Generating Repository layer from $(SQL)..."
	@go run app/generators/main.go repositorygen -sql=$(SQL) -force

# Generate only Store layer
# Usage: make generate-store SQL=schema/users.sql
generate-store:
	@if [ -z "$(SQL)" ]; then \
		echo "❌ Error: SQL parameter is required"; \
		echo "Usage: make generate-store SQL=schema/users.sql"; \
		exit 1; \
	fi
	@echo "🗄️  Generating Store layer from $(SQL)..."
	@go run app/generators/main.go storegen -sql=$(SQL) -force

# Generate only Bridge layer
# Usage: make generate-bridge SQL=schema/users.sql
generate-bridge:
	@if [ -z "$(SQL)" ]; then \
		echo "❌ Error: SQL parameter is required"; \
		echo "Usage: make generate-bridge SQL=schema/users.sql"; \
		exit 1; \
	fi
	@echo "🌉 Generating Bridge layer from $(SQL)..."
	@go run app/generators/main.go bridgegen -sql=$(SQL) -force

# Generate specific layers
# Usage: make generate-layers SQL=schema/users.sql LAYERS=repository,store
generate-layers:
	@if [ -z "$(SQL)" ]; then \
		echo "❌ Error: SQL parameter is required"; \
		echo "Usage: make generate-layers SQL=schema/users.sql LAYERS=repository,store"; \
		exit 1; \
	fi
	@if [ -z "$(LAYERS)" ]; then \
		echo "❌ Error: LAYERS parameter is required"; \
		echo "Usage: make generate-layers SQL=schema/users.sql LAYERS=repository,store"; \
		exit 1; \
	fi
	@echo "🚀 Generating layers [$(LAYERS)] from $(SQL)..."
	@go run app/generators/main.go generate -sql=$(SQL) -layers=$(LAYERS) -force

# Generate from all SQL files in schema/
generate-all-sql:
	@echo "🚀 Generating from all SQL files in schema/..."
	@./scripts/generate.sh all -force

# Generate all code (expand this as you add more generators)
generate: generate-all-sql

# Help for generator commands
generate-help:
	@echo "Generator Makefile Targets:"
	@echo ""
	@echo "Full Stack Generation:"
	@echo "  make generate                      - Generate from all SQL files in schema/"
	@echo "  make generate-sql SQL=<file>       - Generate all layers from specific SQL file"
	@echo "  make generate-all-sql              - Generate from all SQL files"
	@echo ""
	@echo "Individual Layer Generation:"
	@echo "  make generate-repository SQL=<file> - Generate only Repository layer"
	@echo "  make generate-store SQL=<file>      - Generate only Store layer"
	@echo "  make generate-bridge SQL=<file>     - Generate only Bridge layer"
	@echo ""
	@echo "Selective Layer Generation:"
	@echo "  make generate-layers SQL=<file> LAYERS=<layers> - Generate specific layers"
	@echo "    LAYERS options: repository,store,bridge (comma-separated)"
	@echo ""
	@echo "Legacy:"
	@echo "  make generate-pgxstores            - Legacy pgx store generation"
	@echo ""
	@echo "Examples:"
	@echo "  make generate-sql SQL=schema/users.sql"
	@echo "  make generate-repository SQL=schema/users.sql"
	@echo "  make generate-store SQL=schema/users.sql"
	@echo "  make generate-bridge SQL=schema/users.sql"
	@echo "  make generate-layers SQL=schema/users.sql LAYERS=repository,store"
	@echo "  make generate"



dev:
	wgo run app/envoker/main.go
	
dev-admin:
	cd app/envoker/admin/react && pnpm dev
	
tidy:
	go mod tidy
	go mod vendor


generate-pgxstores:
	go generate ./core/repositories/pgxstores


# DATA

dev-data-up:
	docker-compose -f workshop/dev/local-data-compose.yml up  -d

dev-data-down:
	docker-compose -f workshop/dev/local-data-compose.yml down

dev-psql:
	PGPASSWORD=password psql -h localhost -p 5432 -U postgres -d postgres





## --------------------------- RUN

docker-run:
	docker run -p 3000:3000 --env-file .env $(APP_IMAGE_LATEST) 

docker-stop:
	docker stop $(APP_IMAGE_LATEST)

