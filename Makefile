.PHONY: docker-build docker-run docker-push db

# db : Pulls docker image for latest mongo build and runs the container
# docker-build : Builds the docker image for app-api
# docker-push : Push the docker image for app-api to docker hub
# docker-run : Runs the app-api docker image on local machine



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
# With Credentials as given
# To run mongo on shell follow the steps as given below:
# 1. use command `docker ps -l` to list all the containers running on the system
# 2. Search for container running image 'mongo:latest'
# 3. use the container id in the command below to enter mongo shell
# 4. `sudo docker exec -i -t CONTAINER_ID bash`
# 5. use command `exit` to quit mongo shell and docker bash

db:
	mkdir -p ${DB_DATA_DIR}
	docker run \
		-it --rm --net host --name ${DB_CONTAINER_NAME} \
		-v ${PWD}/${DB_DATA_DIR}:/data/db \
		-e MONGO_INITDB_ROOT_USERNAME=${DB_USER} \
		-e MONGO_INITDB_ROOT_PASSWORD=${DB_PASSWORD} \
		mongo:latest
