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
#    template-up
#
#        Deploys Template resource.
#
#        USAGE
#
#            deploy.sh template-up
#
#    up
#
#        Deploys processed template resources for environment.
#
#        USAGE
#
#            deploy.sh up -e ENV [-d]
#
#        OPTIONS
#
#            -d    Dry run
#
#            See COMMON OPTIONS section.
#
#            Note on -e ENV: Value of "prod" will deploy to "api.kscout.io".
#                            Otherwise deploys to "ENV-api.kscout.io".
#
#    down
#
#        Deletes resources for environment.
#
#        USAGE
#
#            deploy.sh down -e ENV
#
#        OPTIONS
#
#            See COMMON OPTIONS section.
#
#    push
#
#        Builds Docker image locally and deploys to an environment.
#
#        USAGE
#
#            deploy.sh push -e ENV
#
#        OPTIONS
#
#            See COMMON OPTIONS section.
#
#    rollout
#
#        Runs "oc rollout ROLLOUT_CMD" on the environment's deployment configuration.
#
#        USAGE
#
#            deploy.sh rollout -e ENV ROLLOUT_CMD
#
#        OPTIONS
#
#            See COMMON OPTIONS section.
#
#        ARGUMENTS
#
#            ROLLOUT_CMD    "oc rollout" sub-command to run.
#
#    labeled
#
#        Runs "oc LABELED_CMD..." with the label option to select resources in the environment.
#
#        USAGE
#
#            deploy.sh labeled -e ENV [-c COMPONENT] LABELED_CMD...
#
#        ARGUMENTS
#
#            LABELED_CMD...    Sub-command to provide to "oc"
#
#        OPTIONS
#
#            -c COMPONENT    Selects with "component" label if provided
#
#            See COMMON OPTIONS section.
#
# COMMON OPTIONS
#
#    The "up", "push", "rollout", and "labeled" commands share the following options:
#
#    -e ENV           Environment
#
#?

# Configuration
set -e

org=kscout
app=serverless-registry-api

image_repo_host=docker.io
image_repo_name="$org/$app"

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
	die "${missing[@]} environment variable(s) required"
    fi
}

# Command argument
cmd="$1"
shift

if [ -z "$cmd" ]; then
    die "CMD argument required"
fi

case "$cmd" in
    template-up)
	bold "Deploying template"

	if ! oc apply -f "$prog_dir/template.yaml"; then
	    die "Failed to deploy template"
	fi

	bold "Done"
	;;
    up)
	# Get options
	while getopts "e:d" opt; do
	    case "$opt" in
		e) env="$OPTARG" ;;
		d) dry_run=true ;;
		?) die "Unknown option" ;;
	    esac
	done

	shift $((OPTIND-1))

	# Checks
	if [ -z "$env" ]; then
	    die "-e ENV option required"
	fi

	# Compute
	image_tag="$env-latest"

	case "$env" in
	    prod) host=api.kscout.io ;;
	    *) host="$env-api.kscout.io" ;;
	esac

	bold "Deploying $env environment"

	# Ensure configuration environment variables are set
	ensure-envs \
	    APP_DB_PASSWORD \
	    APP_GH_INTEGRATION_ID \
	    APP_GH_INSTALLATION_ID \
	    APP_GH_WEBHOOK_SECRET \
	    APP_GH_PRIVATE_KEY_PATH

	# Deploy secrets and config resources
	deploy_cmd="oc apply -f -"
	if [ -n "$dry_run" ]; then
	    deploy_cmd=cat
	fi
	
	cat <<EOF | $deploy_cmd
apiVersion: v1
kind: ConfigMap
metadata:
  name: "$env-mongo-config"
  labels:
    app: serverless-registry-api
    component: mongo
    env: "$env"
data:
  user: "$env-serverless-registry-api"
  dbName: "$env-serverless-registry-api"
---
apiVersion: v1
kind: Secret
metadata:
  name: "$env-mongo-credentials"
  labels:
    app: serverless-registry-api
    component: mongo
    env: "$env"
type: Opaque
data:
  password: $(echo -n "$APP_DB_PASSWORD" | base64)
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: "$env-serverless-registry-api-proxy-config"
  labels:
    app: serverless-registry-api
    component: api
    env: "$env"
data:
  Caddyfile: |
$(cat "$prog_dir/../Caddyfile" | sed 's/^/    /g')
---
apiVersion: v1
kind: Secret
metadata:
  name: "$env-gh-api-configuration"
  labels:
    app: serverless-registry-api
    component: api
    env: "$env"
type: Opaque
data:
  ghIntegrationID: $(echo -n "$APP_GH_INTEGRATION_ID" | base64)
  ghInstallationID: $(echo -n "$APP_GH_INSTALLATION_ID" | base64)
  privateKey: |
$(cat "$APP_GH_PRIVATE_KEY_PATH" | base64 | sed 's/^/    /g')
  webhookSecret: $(echo -n "$APP_GH_WEBHOOK_SECRET" | base64)
EOF
	if [[ "$?" != "0" ]]; then
	    die "Failed to deploy configuration and secrets"
	fi

	# Deploy templated resources
	db_host='$('
	db_host+=$(echo "$env" | tr '[:lower:]' '[:upper:]')
	db_host+=_SERVERLESS_REGISTRY_API_MONGO_SERVICE_HOST
	db_host+=')'
	processed_template=$(oc process "$app" \
				"ENV=$env" \
				"API_IMAGE_SOURCE=$image_repo_host/$image_repo_name" \
				"API_IMAGE_REPOSITORY=$image_repo_name" \
				"API_IMAGE_TAG=$image_tag" \
				"HOST=$host" \
			        "DB_HOST=$db_host")
	if [[ "$?" != "0" ]]; then
	    die "Failed to process template"
	fi

	if ! echo "$processed_template" | $deploy_cmd; then
	    die "Failed to deploy processed template"
	fi

	bold "Done"
	;;
    down)
	# Get options
	while getopts "e:" opt; do
	    case "$opt" in
		e) env="$OPTARG" ;;
		?) die "Unknown option" ;;
	    esac
	done

	shift $((OPTIND-1))

	# Checks
	if [ -z "$env" ]; then
	    die "-e ENV option required"
	fi

	bold "Deleting $env environment"

	# Delete
	for resource in imagestream statefulset pvc deploymentconfig deployment pod route service configmap secret; do
	    echo "- $resource"
	    if ! oc delete "$resource" -l "env=$env,app=$app"; then
		die "Failed to delete $resource"
	    fi
	done
	;;
    push)
	# Get options
	while getopts "e:" opt; do
	    case "$opt" in
		e) env="$OPTARG" ;;
		?) die "Unknown option" ;;
	    esac
	done

	shift $((OPTIND-1))

	# Checks
	if [ -z "$env" ]; then
	    die "-e ENV option required"
	fi

	# Compute
	image_tag="$env-latest"

	case "$env" in
	    prod) host=api.kscout.io ;;
	    *) host="$env-api.kscout.io" ;;
	esac

	bold "Pushing local code to $env environment"

	# Build
	if ! docker build \
	     -t "$image_repo_name:$image_tag" \
	     -t "$image_repo_host/$image_repo_name:$image_tag" \
	     .; then
	    die "Failed to build Docker image"
	fi

	if ! docker push "$image_repo_host/$image_repo_name:$image_tag"; then
	    die "Failed to push Docker image"
	fi

	# Tag image in OpenShift
	if ! oc tag "$image_repo_host/$image_repo_name:$image_tag" \
	     "$image_repo_name:$image_tag"; then
	    die "Failed to tag Docker image in OpenShift"
	fi

	# Wait for image to be present in ImageStream
	image_sha=$(docker inspect \
			   --format='{{ index .RepoDigests 0 }}' \
			   "$image_repo_name:$image_tag")
	if [[ "$?" != "0" ]]; then
	    die "Failed to get Docker image SHA"
	fi

	while true; do
	    if oc describe is "$env-$app" | grep "$image_sha"; then
		echo "Image stream has new Docker image"
		break
	    fi

	    echo "Image stream does not have new Docker image yet..."
	    sleep 1
	done

	bold "Done"
	;;
    rollout)
	# Get options
	while getopts "e:" opt; do
	    case "$opt" in
		e) env="$OPTARG" ;;
		?) die "Unknown option" ;;
	    esac
	done

	shift $((OPTIND-1))

	# Checks
	if [ -z "$env" ]; then
	    die "-e ENV option required"
	fi

	# Arugments
	rollout_cmd="$1"
	shift

	if [ -z "$rollout_cmd" ]; then
	    die "ROLLOUT_CMD argument required"
	fi

	bold "Running: oc rollout $rollout_cmd"

	# Run
	if ! oc rollout "$rollout_cmd" "dc/$env-$app"; then
	    die "Failed to run rollout command"
	fi

	bold "Done"
	;;
    labeled)
	# Get options
	while getopts "e:c:" opt; do
	    case "$opt" in
		e) env="$OPTARG" ;;
		c) component="$OPTARG" ;;
		?) die "Unknown option" ;;
	    esac
	done

	shift $((OPTIND-1))

	# Checks
	if [ -z "$env" ]; then
	    die "-e ENV option required"
	fi

	# Arugments
	labeled_cmd="$@"

	if [ -z "$labeled_cmd" ]; then
	    die "LABELED_CMD argument required"
	fi

	# Compute label option
	label_selectors="env=$env,app=$app"
	if [ -n "$component" ]; then
	    label_selectors+=",component=$component"
	fi

	bold "Running: oc $labeled_cmd -l $label_selectors"

	# Run
	if ! oc $labeled_cmd -l "$label_selectors"; then
	    die "Failed to run labeled command"
	fi

	bold "Done"
	;;
    *) die "CMD must be \"template-up\", \"up\", \"down\", \"push\", \"rollout\", or \"labeled\"" ;;
esac
