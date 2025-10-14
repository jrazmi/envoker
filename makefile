
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

