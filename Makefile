.PHONY: db

DB_DATA_DIR ?= container-data/db
DB_CONTAINER_NAME ?= knative-scout-app-api-db
DB_USER ?= knative-scout-dev
DB_PASSWORD ?= secretpassword

# Start MongoDB server in container
db:
	mkdir -p ${DB_DATA_DIR}
	docker run \
		-it --rm --net host --name ${DB_CONTAINER_NAME} \
		-v ${PWD}/${DB_DATA_DIR}:/data/db \
		-e MONGO_INITDB_ROOT_USERNAME=${DB_USER} \
		-e MONGO_INITDB_ROOT_PASSWORD=${DB_PASSWORD} \
		mongo:latest
