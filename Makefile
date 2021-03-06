.PHONY: push \
	rollout rollout-prod rollout-staging \
	imagestream-tag \
	deploy deploy-prod deploy-staging \
	rm-deploy \
	container container-build container-push \
	db db-cli \
	gh-deploy

MAKE ?= make
CONTAINER_BIN ?= podman

APP ?= serverless-registry-api
CONTAINER_TAG ?= kscout/${APP}:${ENV}-latest

KUBE_LABELS ?= app=${APP},env=${ENV}
KUBE_TYPES ?= dc,configmap,secret,deploy,statefulset,svc,route,is,pod,pvc
KUBE_APPLY ?= oc apply -f -

DB_DATA_DIR ?= container-data/db
DB_CONTAINER_NAME ?= kscout-serverless-registry-api-db
DB_USER ?= kscout-dev
DB_PASSWORD ?= secretpassword

JP ?= jp

# push local code to ENV deploy
push: container imagestream-tag

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

# build and push container image
container: container-build container-push

# build container image for ENV
container-build:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	${CONTAINER_BIN} build -t ${CONTAINER_TAG} .

# push container image for ENV
container-push:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	${CONTAINER_BIN} push ${CONTAINER_TAG}

# start MongoDB server in container
db:
	mkdir -p ${DB_DATA_DIR}
	${CONTAINER_BIN} run \
		-it --rm --net host --name ${DB_CONTAINER_NAME} \
		-v ${PWD}/${DB_DATA_DIR}:/data/db \
		-e MONGO_INITDB_ROOT_USERNAME=${DB_USER} \
		-e MONGO_INITDB_ROOT_PASSWORD=${DB_PASSWORD} \
		mongo:latest

# runs mongo on shell
db-cli:
	container run -it --rm --net host mongo:latest mongo -u ${DB_USER} -p ${DB_PASSWORD}

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
