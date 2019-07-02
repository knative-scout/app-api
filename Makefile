.PHONY: push \
	rollout rollout-prod rollout-staging \
	imagestream-tag \
	deploy deploy-prod deploy-staging \
	rm-deploy \
	docker docker-build docker-push

MAKE ?= make

APP ?= serverless-registry-api
DOCKER_TAG ?= kscout/${APP}:${ENV}-latest

KUBE_LABELS ?= app=${APP},env=${ENV}
KUBE_TYPES ?= dc,configmap,secret,deploy,statefulset,svc,route,is,pod,pvc

KUBE_APPLY ?= oc apply -f -

# push local code to ENV deploy
push: docker imagestream-tag

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
docker: docker-build docker-push

# build docker image for ENV
docker-build:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	docker build -t ${DOCKER_TAG} .

# push docker image for ENV
docker-push:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	docker push ${DOCKER_TAG}
