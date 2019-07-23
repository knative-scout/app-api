#!/usr/bin/env bash
gh_deployment_id_file=/tmp/gh-deployment-id
function die() {
    echo "Error: $@" >&2
    exit 1
}

case "$1" in
    new)
	http post api.github.com/v3/repos/$TRAVIS_REPO_SLUG/deployments \
				'Authorization: bearer $GH_API_TOKEN' \
				ref="$TRAVIS_COMMIT" \
				environment=production \
				description="Travis CI is deploying $TRAVIS_REPO_SLUG" > "$gh_deployment_id_file"
	if [[ "$?" != "0" ]]; then
	    die "Failed to create GitHub deployment"
	fi
	;;
    set-state)
	state="$2"
	if [ -z "$state" ]; then
	    die "EXTRA argument required to be state of deployment status"
	fi

	case "$state" in
	    ^error|failure|inactive|in_progress|queued|pending|success)
		echo "\"$state\" is OK"
		;;
	    *)
		die "Invalid state \"$state\""
		;;
	esac
	

	if [ ! -f "$gh_deployment_id_file" ]; then
	    die "\"$gh_deployment_id_file\" must exist"
	fi

	gh_deployment_id=$(cat "$gh_deployment_id")
	if [[ "$?" != "0" ]]; then
	    die "Failed to read GitHub deployment ID"
	fi

	http post api.github.com/v3/repos/$TRAVIS_REPO_SLUG/deployments/$gh_deployment_id/statuses \
	     'Authorization: bearer $GH_API_TOKEN' \
	     'Accept: application/vnd.github.ant-man-preview+json' \
	     'Accept: application/vnd.github.flash-preview+json' \
	     log_url="https://travis-ci.org/$TRAVIS_REPO_SLUG/builds/$TRAVIS_BUILD_ID" \
	     environment=production \
	     description="Travis CI is in $state" \
	     environment_url="api.kscout.io"
	if [[ "$?" != "0" ]]; then
	    die "Failed to set GitHub deployment status"
	fi
	;;
    *)
	progname='gh-deploy-status.sh'
	cat <<EOF
$progname - Set GitHub deployment statuses
See GitHub API docs: https://developer.github.com/v3/repos/deployments/#

Usage: $progname CMD [EXTRA]

Arguments:
1. CMD (String): One of:
   - new: creates new deployment and store id in "$gh_deploy_id_file"
   - set-state: sets deployment status to EXTRA argument
2. EXTRA (Optional, any type): Extra arguments passed to some commands
EOF
	die "Unknown command \"$1\""
	;;
esac

