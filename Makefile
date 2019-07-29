.PHONY: push \
	rollout rollout-prod rollout-staging \
	imagestream-tag \
	deploy deploy-prod deploy-staging \
	rm-deploy \
	docker docker-build docker-push \
	db db-cli \
	ci-deploy \
	gh-deploy

MAKE ?= make

APP ?= serverless-registry-api
DOCKER_TAG ?= kscout/${APP}:${ENV}-latest

KUBE_LABELS ?= app=${APP},env=${ENV}
KUBE_TYPES ?= dc,configmap,secret,deploy,statefulset,svc,route,is,pod,pvc
KUBE_APPLY ?= oc apply -f -

DB_DATA_DIR ?= container-data/db
DB_CONTAINER_NAME ?= kscout-serverless-registry-api-db
DB_USER ?= kscout-dev
DB_PASSWORD ?= secretpassword

JP ?= jp

# push local code to ENV deploy
push: docker imagestream-tag

# rollout ENV
rollout:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	oc rollout latest dc/${ENV}-${APP}

# rollout production
rollout-prod:
	${MAKE} rollout ENV=prod
	${MAKE} gh-deploy ENV=production

# rollout staging
rollout-staging:
	${MAKE} rollout ENV=staging
	${MAKE} gh-deploy ENV=staging

# import latest tag for ENV to imagestream
imagestream-tag:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	oc tag docker.io/kscout/${APP}:${ENV}-latest ${ENV}-${APP}:${ENV}-latest --scheduled

# deploy to ENV
deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	helm template \
		--values deploy/values.yaml \
		--values deploy/values.secrets.${ENV}.yaml \
		--set global.env=${ENV} deploy \
	| ${KUBE_APPLY}

# deploy to production
deploy-prod:
	${MAKE} deploy ENV=prod

# deploy to staging
deploy-staging:
	${MAKE} deploy ENV=staging

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

# build docker image for ENV
docker-build:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	docker build -t ${DOCKER_TAG} .

# push docker image for ENV
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

# deploy app via CI, creates GitHub deployment statuses to track progress
ci-deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	./deploy/gh-deploy-status.sh new
	./deploy/gh-deploy-status.sh set-state in_progress
	${MAKE} docker
	${MAKE} deploy-prod
	./deploy-gh-deploy-status.sh set-state success

# create deployment status for current commit
# The jp command is required.
# STATE is the state of the deployment status, can be error, failure, inactive, queued, or success. Defaults to success.
# REF is the GitHub reference of the code which is deployed, defaults to local HEAD's sha.
# ENV is the environment. Defaults to production
gh-deploy:
	if [ -z "${MAKEFILE_GH_API_TOKEN}" ]; then echo "MAKEFILE_GH_API_TOKEN must be set" >&2; exit 1; fi

	$(eval STATE ?= success)
	$(eval REF ?= $(shell git rev-parse HEAD))
	$(eval ENV ?= production)

	$(eval id ?= $(shell curl -X POST -H "Authorization: bearer ${MAKEFILE_GH_API_TOKEN}" -d "{\"ref\": \"${REF}\", \"environment\": \"${ENV}\"}" "https://api.github.com/repos/kscout/${APP}/deployments" | ${JP} id))
	curl -X POST -H "Authorization: bearer ${MAKEFILE_GH_API_TOKEN}" -d "{\"state\": \"${STATE}\"}" "https://api.github.com/repos/kscout/${APP}/deployments/${id}/statuses"

