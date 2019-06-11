#!/usr/bin/env bash
#?
# deploy.sh - Deploy API and database
#
# USAGE
#
#    deploy.sh [-e ENV] [-h HOST] [CMD]
#
# OPTIONS
#
#    -s SECRETS_F    File which contains secret data, see SECRETS section for required keys
#    -e ENV     (Optional) Deployment environment, defaults to "prod"
#    -h HOST    (Optional) API host. If ENV == prod defaults to "api.kscout.io".
#                          If ENV is anything else defaults to "${ENV}-api.kscout.io"
#
# ARGUMENTS
#
#    CMD    (Optional) Kubectl command, defaults to "apply"
#
# SECRETS
#
#    Secrets will be provided via JSON / YAML / TOML file with the following structure:
#
#    - mongo (Object): MongoDB secrets
#      - password (String): API user password
#    - github (Object): GitHub secrets
#      - apiToken (String): API token with which to make authenticated API calls
#      - webhookSecret (String): Secret GitHub uses to sign an HMAC of requests sent
#                                to the API webhook endpoint
#
# BEHAVIOR
#
#    Runs gomplate (https://docs.gomplate.ca/) over the resource files in the
#    deploy/templates directory. Resource files can use Go templating. The following
#    keys will hold data:
#
#    - .config: Will hold values of -e and -h options in object with keys "env" and "host"
#    - .secrets: Holds secret values, see SECRETS section for structure of object
#
#    Deploys the resulting YAML to Kubernetes under the kscout namespace.
#
#?

prog_dir=$(realpath $(dirname "$0")) 

# {{{1 Helpers
function die() {
    echo "Error: $@" >&2
    exit 1
}

function bold() {
    echo "$(tput bold)$@$(tput sgr0)"
}

# {{{1 Options
# {{{2 Defaults
env=prod
host=api.kscout.io

# {{{2 Get
while getopts "s:e:h:" opt; do
    case "$opt" in
	e)
	    env="$OPTARG"
	    ;;
	h)
	    host="$OPTARG"
	    custom_host=true
	    ;;
	s)
	    secrets_file="$(realpath $OPTARG)"
	    ;;
	'?') die "Unknown option" ;;
    esac
done

shift $(expr $OPTIND - 1)

# {{{2 Computed defaults
if [ -z "$custom_host" ] && [[ "$env" != "prod" ]]; then
    host="$env-api.kscout.io"
fi

# {{{2 Ensure -s option provided
if [ -z "$secrets_file" ]; then
    die "-s SECRETS_F option must be provided"
fi

if [ ! -f "$secrets_file" ]; then
    die "-s $secrets_file file does not exist"
fi

secrets_file_ext=${secrets_file##*.}

# {{{1 Arguments
cmd="$1"

if [ -z "$cmd" ]; then
    cmd=apply
fi

# {{{1 Print run parameters
bold "Configuration"
echo "env : $env"
echo "host: $host"

# {{{1 Bake templates
bold "Baking resource templates"
# {{{2 Save input option data as file
sha=$(echo "$data_file_contents" | sha256sum | awk '{ print $1 }')

data_file_contents="{\"env\": \"$env\", \"host\": \"$host\"}"
data_file="/tmp/$sha.json"

out_dir="/tmp/$sha-out"

mkdir -p "$out_dir"
echo "$data_file_contents" > "$data_file"

function cleanup() {
    if [ -f "$data_file" ]; then
	if ! rm "$data_file"; then
	    die "Failed to delete data file: $data_file"
	fi
    fi

    if [ -d "$out_dir" ]; then
	if ! rm -rf "$out_dir"; then
	    die "Failed to delete out dir: $out_dir"
	fi
    fi
}

trap cleanup EXIT

# {{{2 Run
docker run \
       -it \
       --rm \
       -v "$prog_dir/templates:/in" \
       -v "$out_dir:/out" \
       -v "$data_file:/tmp/data-file.json" \
       -v "$secrets_file:/tmp/secrets-file.$secrets_file_ext" \
       noahhuppert/gomplate:dev \
       --input-dir /in \
       --output-dir /out \
       -c config=/tmp/data-file.json \
       -c secrets=/tmp/secrets-file."$secrets_file_ext"

if [[ "$?" != "0" ]]; then
    die "Failed to bake templates"
fi

# {{{2 Deploy
bold "kubectl -n kscout $cmd"
for file in $(ls "$out_dir"); do
    cat "$out_dir/$file" | kubectl -n kscout "$cmd" --filename -
done

