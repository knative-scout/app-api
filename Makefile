.PHONY: docker docker-build docker-run docker-push db db-cli


DB_DATA_DIR ?= container-data/db
DB_CONTAINER_NAME ?= kscout-serverless-registry-api-db
DB_USER ?= kscout-dev
DB_PASSWORD ?= secretpassword

DOCKER_TAG_VERSION ?= dev-latest
DOCKER_TAG ?= kscout/serverless-registry-api:${DOCKER_TAG_VERSION}

# builds and pushes a docker image
docker: docker-build docker-push

# build Docker image
docker-build:
	docker build -t ${DOCKER_TAG} .


# Push to docker hub
docker-push:
	docker push ${DOCKER_TAG}


# Runs the docker image locally
docker-run:
	docker run -it --rm -e APP_GH_TOKEN=${APP_GH_TOKEN} -e APP_GH_WEBHOOK_SECRET=${APP_GH_WEBHOOK_SECRET} --net host ${DOCKER_TAG}


# Start MongoDB server in container
db:
	mkdir -p ${DB_DATA_DIR}
	docker run \
		-it --rm --net host --name ${DB_CONTAINER_NAME} \
		-v ${PWD}/${DB_DATA_DIR}:/data/db \
		-e MONGO_INITDB_ROOT_USERNAME=${DB_USER} \
		-e MONGO_INITDB_ROOT_PASSWORD=${DB_PASSWORD} \
		mongo:latest

# Runs mongo on shell
db-cli:
	docker run -it --rm --net host mongo:latest mongo -u ${DB_USER} -p ${DB_PASSWORD}
