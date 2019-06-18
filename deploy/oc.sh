#!/usr/bin/env bash
#?
# oc.sh - Runs the oc command with the labels argument
#
# USAGE
#
#    oc.sh -e ENV [-c COMPONENT] CMD...
#
# OPTIONS
#
#    -e ENV          Deployment environment
#    -c COMPONENT    Component label value, defaults to "api"
#
# ARGUMENTS
#
#    CMD...       oc command and arguments to run
#
# BEHAVIOR
#
#    Runs oc CMD... with the following label selector:
#
#    -l app=serverless-registry-api,component=COMPONENT,env=ENV
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
while getopts "e:c:" opt; do
    case "$opt" in
	e) env="$OPTARG" ;;
	c) component="$OPTARG" ;;
	?) die "Unknown option" ;;
    esac
done

if [ -z "$env" ]; then
    die "-e ENV option required"
fi

if [ -z "$component" ]; then
    component=api
fi

shift $(expr $OPTIND - 1)

# Run
oc "$@" -l "env=$env,app=$app,component=$component"
