#!/usr/bin/env bash
#?
# deploy.sh - Deployment management
#
# USAGE
#
#    deploy.sh CMD OPTIONS
#
# ARGUMENTS
#
#    CMD    Command to run, see COMMANDS section
#
# COMMANDS
#
#    template
#
#        Deploys Template resource.
#
#    up
#
#        Deploys processed template resources for environment.
#
#        OPTIONS
#
#            See COMMON OPTIONS section.
#
#    build
#
#        Builds Docker image locally and pushes to integrated cluster registry.
#
#        OPTIONS
#
#            See COMMON OPTIONS section.

# COMMON OPTIONS
#
#    The "up" and "build" commands share the following options:
#
#    -e ENV           Environment.
#    -r IMAGE_REPO    
#    -t IMAGE_TAG     Name of Docker image tag. Defaults to "$env-latest".
#
#?

# Configuration
app=serverless-registry-api

default_image_repo="quay.io/kscout/$app"
default_image_repo_staging="kscout/$app"

# Helpers
prog_dir=$(realpath $(dirname "$0"))

function bold() {
    echo "$(tput bold)$@$(tput sgr0)"
}

function die() {
    echo "Error: $@" >&2
    exit 1
}

function ensure-envs() {
    missing=()
    for name in "$@"; do
	if [ -z "${!name}" ]; then
	    missing+=("$name")
	fi
    done

    if [ -n "$missing" ]; then
	die "${missing[@]} environment variables required"
    fi
}

function get-common-options() {
    # Get
    while getopts "e:r:t:" opt; do
	case "$opt" in
	    e) env="$OPTARG" ;;
	    r) image_repo="$OPTARG" ;;
	    t) image_tag="$OPTARG" ;;
	    ?) die "Unknown option" ;;
	esac
    done

    # Checks
    if [ -z "$env" ]; then
	    die "-e ENV option required"
    fi    

    # Defaults
    if [ -z "$image_repo" ]; then
	case "$env" in
	    staging) image_repo="$default_image_repo_staging" ;;
	    *) image_repo="$default_image_repo" ;;
	esac
    fi

    if [ -z "$image_tag" ]; then
	image_tag="$env-latest"
    fi
}

# Command argument
cmd="$1"
shift

if [ -z "$cmd" ]; then
    die "CMD argument required"
fi

case "$cmd" in
    template)
	bold "Deploying template"

	if ! oc apply -f "$prog_dir/template.yaml"; then
	    die "Failed to deploy template"
	fi

	bold "Done"
	;;
    up)
	# Get options
	get-common-options

	bold "Deploying $env environment"

	# Ensure configuration environment variables are set
	ensure-envs \
	    APP_DB_PASSWORD \
	    APP_GH_INTEGRATION_ID \
	    APP_GH_INSTALLATION_ID \
	    APP_GH_WEBHOOK_SECRET \
	    APP_GH_PRIVATE_KEY_PATH

	# Deploy secrets and config resources
	cat <<EOF | oc apply -f -
        apiVersion: v1
	kind: ConfigMap
	metadata:
	  name: $env-mongo-config
	  labels:
	    app: serverless-registry-api
	    component: mongo
	    env: $env
	data:
	  user: $env-serverless-registry-api
	  dbName: $env-serverless-registry-api
        ---
	apiVersion: v1
	kind: Secret
	metadata:
	  name: $env-mongo-credentials
	  labels:
	    app: serverless-registry-api
	    component: mongo
	    env: $env
	type: Opaque
	data:
	  password: $(echo "$APP_DB_PASSWORD" | base64)
        ---
        apiVersion: v1
	kind: Secret
	metadata:
	  name: $env--gh-api-configuration
	  labels:
	    app: serverless-registry-api
	    component: api
	    env: $env
	type: Opaque
	data:
	  ghIntegrationID: $(echo "$APP_GH_INTEGRATION_ID" | base64)
	  ghInstallationID: $(ehco "$APP_GH_INSTALLATION_ID" | base64)
	  privateKey: $(cat "$APP_GH_PRIVATE_KEY_PATH" | base64")
	  webhookSecret: $(echo "$APP_GH_WEBHOOK_SECRET | base64)
EOF
	if [[ "$?" != "0" ]]; then
	    die "Failed to deploy configuration and secrets"
	fi

	# Deploy templated resources
	processed_template=$(oc process "$app" \
				"ENV=$env" \
				"API_IMAGE_REPOSITORY=$image_repository" \
				"API_IMAGE_TAG=$image_tag")
	if [[ "$?" != "0" ]]; then
	    die "Failed to process template"
	fi

	if ! echo "$processed_template" | oc apply -f -; then
	    die "Failed to deploy processed template"
	fi

	bold "Done"
	;;
    build)
	# Get options
	get-common-options

	bold "Building"

	if ! docker build -t "kscout/$app:$image_tag" .; then
	    die "Failed to build"
	fi

	if ! docker push
