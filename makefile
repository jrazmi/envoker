SHELL_PATH := /bin/ash
SHELL := $(if $(wildcard $(SHELL_PATH)),/bin/ash,/bin/bash)

GOLANG := golang:1.24.1

NAMESPACE := envoker
BASE_IMAGE_NAME := localhost/envoker
VERSION := 0.0.1

ENVOKER_APP := envoker
ENVOKER_IMAGE_NAME := $(BASE_IMAGE_NAME)/$(ENVOKER_APP):$(VERSION)
ENVOKER_IMAGE_LATEST := $(BASE_IMAGE_NAME)/$(ENVOKER_APP):latest

# ---------------- BUILD
build: build-envoker

build-envoker:
	docker build \
    -f workshop/docker/dockerfile.envoker\
    -t $(ENVOKER_IMAGE_NAME) \
    -t $(ENVOKER_IMAGE_LATEST) \
    --build-arg BUILD_REF=$(VERSION) \
    --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    .

dev:
	wgo run app/envoker/main.go
	
dev-admin:
	cd app/envoker/admin/react && pnpm dev
	
tidy:
	go mod tidy
	go mod vendor





# DATA

dev-data-up:
	docker-compose -f workshop/dev/local-data-compose.yml up  -d

dev-data-down:
	docker-compose -f workshop/dev/local-data-compose.yml down

dev-psql:
	PGPASSWORD=password psql -h localhost -p 5432 -U postgres -d postgres





## --------------------------- RUN

run-latest:
	docker run -p 3000:3000 --env-file .env $(ENVOKER_IMAGE_LATEST) 

stop-latest:
	docker stop $(ENVOKER_IMAGE_LATEST)

