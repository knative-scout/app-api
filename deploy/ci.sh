#!/usr/bin/env bash
#?
# ci.sh - Test, build, deploy
#
# USAGE
#
#    ci.sh
#
# BEHAVIOR
#
#    Run by Travis CI.
#
#?

set -e

# Helpers
prog_dir=$(realpath $(dirname "$0"))
repo_dir=$(realpath "../$prog_dir")

function die() {
    echo "Error: $@" >&2
    exit 1
}

function bold() {
    echo "$(tput bold)$@$(tput sgr0)"
}

# Configuration
export ENV="$TRAVIS_COMMIT"
export DOCKER_VERSION="$TRAVIS_COMMIT"
gh_deploy_flag_f=/tmp/gh-deploy-created

# Test
bold "Testing"

make -C "$repo_dir" test

# Build
bold "Building Docker image"

make -C "$repo_dir" docker

# Deploy
bold "Deploying"

function cleanup_deploy() {
    # Create GitHub deployment status if deployment was created
    if [ -f "$gh_deploy_flag_f" ]; then
	   if [[ "$?" == "0" ]]; then
		  "$prog_dir/gh-deploy-status.sh" set-status success
	   else
		  "$prog_dir/gh-deploy-status.sh" set-status failure
	   fi
    fi
}
trap cleanup_deploy EXIT

"$prog_dir/gh-deploy-status.sh" new && touch "$gh_deploy_flag_f"

make -C "$repo_dir" deploy
