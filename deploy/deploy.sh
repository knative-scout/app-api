#!/usr/bin/env bash
#?
# deploy.sh - Manage deployments
#
# USAGE
#
#    deploy.sh CMD COMPONENT
#
# ARGUMENTS
#
#    CMD          Action to complete, valid values: up, down
#    COMPONENT    Part of deployment to manage, valid values: app, db, all
#
# BEHAVIOR
#
#    The up command deploys resources, the down command destroyes resources.
#
#    The app component refers to the API server. The db component refers to the
#    MongoDB server. The all component refers to both.
#
#?

# {{{1 Configuration
prog_dir=$(realpath $(dirname "$0"))
kubectl="$prog_dir/../tmpk"

common_file_args=(
    "--filename" "$prog_dir/ns.yaml"
    "--filename" "$prog_dir/secrets.yaml"
)
app_file_args=(
    "--filename" "$prog_dir/app.yaml"
)
db_file_args=(
    "--filename" "$prog_dir/db.yaml"
)

# {{{1 Helpers
function die() {
    echo "Error: $@" >&2
    exit 1
}

function bold() {
    echo "$(tput bold)$@$(tput sgr0)"
}

# {{{1 Arguments
cmd="$1"
component="$2"

case "$cmd" in
    up) kubectl_cmd=apply ;;
    down) kubectl_cmd=delete ;;
    *) die "CMD arugment must be \"up\" or \"down\"" ;;
esac

file_args=("${common_file_args[@]}")
case "$component" in
    app) file_args+=("${app_file_args[@]}") ;;
    db) file_args+=("${db_file_args[@]}") ;;
    all) file_args+=("${app_file_args[@]}" "${db_file_args[@]}") ;;
    *) die "COMPONENT arugment must be \"app\", \"db\", or \"all\"" ;;
esac

# {{{1 Run
bold "Bringing $component $cmd"

if ! $kubectl "$kubectl_cmd" "${file_args[@]}"; then
    die "Failed to bring $component $cmd"
fi

bold "Success"
