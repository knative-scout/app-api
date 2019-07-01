.PHONY: rollout rollout-prod rollout-staging \
	deploy-yaml deploy deploy-prod deploy-staging \
	rm-deploy \
	docker docker-build docker-push

MAKE ?= make

APP ?= serverless-registry-api
DOCKER_TAG ?= kscout/${APP}:${ENV}-latest

KUBE_LABELS ?= app=${APP},env=${ENV}
KUBE_TYPES ?= dc,configmap,secret,deploy,statefulset,svc,route,is,pod,pv,pvc

# rollout ENV
rollout:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	oc rollout latest dc/${ENV}-${APP}

# rollout production
rollout-prod:
	${MAKE} rollout ENV=prod

# rollout staging
rollout-staging:
	${MAKE} rollout ENV=staging

# display YAML resource definitions for ENV
deploy-yaml:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	@helm template \
		--values deploy/values.yaml \
		--values deploy/values.secrets.${ENV}.yaml \
		--set global.env=${ENV} deploy

# deploy to ENV
deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	helm template \
		--values deploy/values.yaml \
		--values deploy/values.secrets.${ENV}.yaml \
		--set global.env=${ENV} deploy \
	| oc apply -f -

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
	oc get -l ${KUBE_LABELS} ${KUBE_TYPES} -o yaml | oc delete -f -

# build and push docker image
docker: docker-build docker-push

# build docker image for ENV
docker-build:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	docker build -t ${DOCKER_TAG} .

# push docker image for ENV
docker-push:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	docker push ${DOCKER_TAG}
