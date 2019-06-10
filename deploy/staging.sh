#!/usr/bin/env bash
#?
# staging.sh - Staging management script
#
# USAGE
#
#    staging.sh CMD
#
# ARGUMENTS
#
#    CMD    Action to complete, can be "rollout" or "logs"
#
# ROLLOUT COMMAND
#
#    Builds local code into a Docker image with the "staging-latest" tag. Pushes to
#    Docker hub.
#
#    Refreshes the "staging-latest" tag on OpenShift.
#
#    Triggers a deployment of the new tag.
#
#    Waits for deployment to complete.
#
# LOGS COMMAND
#
#    Displays logs for current staging pods.
#
#?

# {{{1 Configuration
prog_dir=$(realpath $(dirname "$0"))

docker_tag_version=staging-latest
docker_repo=kscout/app-api
docker_tag="$docker_repo:$docker_tag_version"

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

if [ -z "$cmd" ]; then
    die "CMD argument must be provided"
fi

case "$cmd" in
    rollout)
	# {{{1 Build Docker
	bold "Building Docker $docker_tag"

	if ! docker build -t "$docker_tag" .; then
	    die "Failed to build $docker_tag"
	fi

	# {{{1 Push Docker
	bold "Pushing Docker $docker_tag"

	if ! docker push "$docker_tag"; then
	    die "Failed to push $docker_tag"
	fi

	# {{{1 Refresh tag on OpenShift
	bold "Refreshing $docker_tag on OpenShift"

	if ! oc tag "docker.io/$docker_tag" "$docker_tag" --scheduled; then
	    die "Failed to refresh $docker_tag on OpenShift"
	fi

	# {{{1 Wait for image stream to have new tag
	image_sha=$(docker inspect --format='{{ index .RepoDigests 0 }}' "$docker_tag")

	if [[ "$?" != "0" ]]; then
	    die "Failed to get Sha of $docker_tag"
	fi

	while true; do
	    if oc describe is app-api | grep "$image_sha"; then
		echo "Image stream has new $docker_tag"
		break
	    fi

	    echo "Image stream does not have new $docker_tag yet..."
	    sleep 1
	done

	# {{{1 Roll out new docker tag
	bold "Rolling out new $docker_tag"

	if ! oc rollout latest dc/staging-app-api; then
	    die "Failed to rollout new $docker_tag"
	fi

	# {{{1 Wait for rollout to complete
	bold "Waiting for rollout of $docker_tag to complete"

	if ! oc rollout status dc/staging-app-api; then
	    die "Failed to wait for rollout $docker_tag"
	fi

	bold "Done"
	;;
    logs)
	bold "Viewing staging logs"
	kubectl logs -f -l app=app-api,env=staging
	;;
    *)
	die "Unknown command $cmd"
esac
