.PHONY: deploy deploy-prod deploy-staging rm-deploy docker docker-build docker-push

MAKE ?= make

DOCKER_TAG ?= kscout/serverless-registry-api:${ENV}-latest

KUBE_LABELS ?= app=serverless-registry-api,env=${ENV}
KUBE_TYPES ?= dc,configmap,secret,deploy,svc,route,is,pv,pvc

# deploy to ENV
deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	helm template --values deploy/values.yaml --set global.env=${ENV} deploy | oc apply -f -

# deploy to production
deploy-prod:
	${MAKE} deploy ENV=prod

# deploy to staging
deploy-staging:
	${MAKE} deploy ENV=staging

# remove deployment for ENV
rm-deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
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
