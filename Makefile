.PHONY: docker-build docker-run docker-push db

DB_DATA_DIR ?= container-data/db
DB_CONTAINER_NAME ?= knative-scout-app-api-db
DB_USER ?= knative-scout-dev
DB_PASSWORD ?= secretpassword

DOCKER_TAG ?= kscout/app-api:dev-latest

# build Docker image
docker-build:
	docker build -t ${DOCKER_TAG} .

# push Docker image to Docker hub
docker-push:
	docker push ${DOCKER_TAG}

# run Docker container
docker-run:
	docker run -it --rm --net host ${DOCKER_TAG}

# Start MongoDB server in container
db:
	mkdir -p ${DB_DATA_DIR}
	docker run \
		-it --rm --net host --name ${DB_CONTAINER_NAME} \
		-v ${PWD}/${DB_DATA_DIR}:/data/db \
		-e MONGO_INITDB_ROOT_USERNAME=${DB_USER} \
		-e MONGO_INITDB_ROOT_PASSWORD=${DB_PASSWORD} \
		mongo:latest
