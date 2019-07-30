.PHONY: deploy rm-deploy \
	docker docker-build docker-push \
	db db-cli \
	test

MAKE ?= make

ORG ?= kscout
APP ?= serverless-registry-api

DOCKER_VERSION ?= ${ENV}-latest
DOCKER_TAG ?= ${ORG}/${APP}:${DOCKER_VERSION}

KUBE_LABELS ?= app=${APP},env=${ENV}
KUBE_TYPES ?= $(shell grep -h -r kind deploy/charts | awk '{ print $2 }' | uniq | paste -sd "," -)
KUBE_APPLY ?= oc apply -f -

DB_DATA_DIR ?= container-data/db
DB_CONTAINER_NAME ?= kscout-serverless-registry-api-db
DB_USER ?= kscout-dev
DB_PASSWORD ?= secretpassword

# deploy to ENV
deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	helm template \
		--values deploy/values.yaml \
		--values deploy/values.secrets.${ENV}.yaml \
		--set global.env=${ENV} \
		--set http.appImage=docker.io/${DOCKER_TAG} \
		${SET_ARGS} deploy \
	| ${KUBE_APPLY}
	oc rollout status "dc/${ENV}-${APP}"

# remove deployment for ENV
rm-deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	@echo "Remove ${ENV} ${APP} deployment"
	@echo "Hit any key to confirm"
	@read confirm
	oc get \
		--ignore-not-found \
		-l ${KUBE_LABELS} \
		${KUBE_TYPES} \
		-o yaml \
	| oc delete -f -

# build and push docker image
docker:
	@if [ -eq "$LOGIN" "true" ]; then echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin &> /dev/null
	${MAKE} docker-build
	${MAKE} docker-push

# build docker image
docker-build:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	docker build -t ${DOCKER_TAG} .

# push docker image
docker-push:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	docker push ${DOCKER_TAG}

# start MongoDB server in container
db:
	mkdir -p ${DB_DATA_DIR}
	docker run \
		-it --rm --net host --name ${DB_CONTAINER_NAME} \
		-v ${PWD}/${DB_DATA_DIR}:/data/db \
		-e MONGO_INITDB_ROOT_USERNAME=${DB_USER} \
		-e MONGO_INITDB_ROOT_PASSWORD=${DB_PASSWORD} \
		mongo:latest

# runs mongo on shell
db-cli:
	docker run -it --rm --net host mongo:latest mongo -u ${DB_USER} -p ${DB_PASSWORD}

# test app
test:
	go test ./...
