#!/usr/bin/env bash
#?
# template.sh - Outputs a processed template
#
# USAGE
#
#    template.sh -e ENV
#
# OPTIONS
#
#    -e ENV    Deployment environment
#
#?

# Helpers
function die() {
    echo "Error: $@" >&2
    exit 1
}

# Configuration
app=serverless-registry-api

# Options
while getopts "e:" opt; do
    case "$opt" in
	e) env="$OPTARG" ;;
	?) die "Unknown option" ;;
    esac
done

if [ -z "$env" ]; then
    die "-e ENV option required"
fi

# Check if env vars for template parameters are set
required_env_vars=(APP_DB_PASSWORD
		   APP_GH_INTEGRATION_ID
		   APP_GH_INSTALLATION_ID
		   APP_GH_WEBHOOK_SECRET
		   APP_GH_PRIVATE_KEY_PATH)
for var_name in ${required_env_vars[@]}; do
    if [ -z "${!var_name}" ]; then
	die "$var_name environment variable must be set"
    fi
done

# Build template parameters
template_prameters=()

template_parameters+=("ENV=$env"
		      "UPPER_ENV=$(echo -n $env | tr '[:lower:]' '[:upper:]')")

if [[ "$env" == "prod" ]]; then
    host="api.kscout.io"
else
    host="$env-api.kscout.io"
fi
template_parameters+=("HOST=$host")

template_parameters+=(B64_MONGO_PASSWORD=$(echo -n "$APP_DB_PASSWORD" | base64))
template_parameters+=(B64_GH_INTEGRATION_ID=$(echo -n "$APP_GH_INTEGRATION_ID" | base64))
template_parameters+=(B64_GH_INSTALLATION_ID=$(echo -n "$APP_GH_INSTALLATION_ID" | base64))

if [ ! -f "$APP_GH_PRIVATE_KEY_PATH" ]; then
    die "Private key \"$APP_GH_PRIVATE_KEY_PATH\" does not exist"
fi
template_parameters+=(B64_GH_PRIVATE_KEY=$(cat "$APP_GH_PRIVATE_KEY_PATH" | base64))
template_parameters+=(B64_GH_WEBHOOK_SECRET=$(echo -n "$APP_GH_WEBHOOK_SECRET" | base64))

# Output template
oc process "$app" "${template_parameters[@]}"
