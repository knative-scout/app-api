# Design
API design.

# Table Of Contents
- [Overview](#overview)
- [Data Model](#data-model)
- [Endpoints](#endpoints)
- [Deployment Script](#deployment-script)

# Overview
HTTP RESTful API.  

Requests pass data via JSON encoded bodies except for in GET requests where data
will be passed via URL and query parameters.

Responses will always return JSON.

# Data Model
## App Model
`apps` collection.

[Documentation of schema fields](https://godoc.org/github.com/kscout/serverless-registry-api/models#App)

Schema:

- `app_id` (String)
- `name` (String)
- `tagline` (String)
- `description` (String)
- `screenshot_urls` (List[String])
- `logo_url` (String)
- `tags` (List[String])
- `verification_status` (String): One of `pending`, `verifying`, `good`, `bad`
- `github_url` (String)
- `deployment` (Object): Deployment details. Has keys:
  - `resources` (List[String]): Each list entry holds the JSON for one
	deployment resource
  - `parameterized_resources` (List[String]): Same as `resources` except
	values in `ConfigMap` and `Secret` resources are replaced with bash 
	variable names from `parameters`
  - `parameters` (List[Object]): Information about parameters in
	`parameterized_resources`, has keys:
     - `substitution` (String): Value which should be substituted for
	   actual value
	 - `display_name` (String): Name of variable to display to user
	 - `default_value` (String): Value that existed before was parameterized
	 - `requires_base64` (Boolean): Indicates if the value must be base64 
		 encoded in the template
  - `deploy_script` (String): Custom Bash deploy script for app		 
- `version` (String)
- `author` (String)
- `maintainer` (String)

## Resource Model
`resource` collection.

Schema:

- `resource_id` (String)
- `name` (String)
- `description` (String)
- `author` (String)

# Endpoints
Most endpoints to not require authentication.  

Those which do will be marked. Provide authentication as a bearer token in the
`Authorization` header.  

Endpoints which specify a response of `None` will return the 
JSON: `{"ok": true}`.

## App Endpoints
### Search Apps
`GET /apps?query=<query>&tags=<tags>&categories=<categories>`

Search serverless apps in hub.

If no search parameters are provided all applications will be returned.

Request:

- `query` (Optional, String): Search Keywords
- `tags` (Optional, List[String]): Tags applications must have
- `categories` (Optional, List[String]): Categories applications must be part of

Response:

- `apps` (List[[App Model](#app-model)])

### Natural Search
`GET /nsearch?query=<query>`

Search serverless apps in hub using natural language query.

If no search parameters are provided all applications will be returned.

**Exclude words from your search**<br/>
Put - in front of a word you want to leave out. For example, python app -flask

**Search for an exact match**<br/>
Put a word or phrase inside quotes. For example, "nodejs" app


Request:

- `query` (Optional, String): Natural Language Query

Response:

- `apps` (List[[App Model](#app-model)])



### Get App By ID
`GET /apps/id/<app_id>`

Get application by ID.

Request:

- `app_id` (String)

Response:

- `app` ([App Model](#app-model))

### App Pull Request Webhook Endpoint
`POST /apps/webhook`

GitHub will make a request to this endpoint every time a new pull request is 
made to submit an app.

Request:

- [GitHub Webhook Request](https://developer.github.com/webhooks/#payloads)
- [`PullRequestEvent'](https://developer.github.com/v3/activity/events/types/#pullrequestevent)

Response: None

### Search Tags
`GET /apps/tags?query=<query>`

Get all available tags.

Request:

- `query` (Optional, String): Search string, if empty all tags will be returned

Response:

- `tags` (List[String])

### Search Categories
`GET /apps/categories?query=<query>`

Get all available categories.

Request:

- `query` (Optional, String): Search string, if empty all categories will 
  be returned

Response:

- `categories` (List[String])

### Get Deployment File
`GET /apps/<app_id>/version/<version>/deployment.json`  

Get file with all an app's deployment resources.

Request:

- `app_id` (String): ID of app
- `version` (String): Version of app

Response: JSON text of deployment resources

### Get Deployment Script
`GET /apps/<app_id>/version/<version>/deploy.sh`

Get deployment script for version of app.  
See [deployment script](#deployment-script) for design details.

Request:

- `app_id` (String): ID of app
- `version` (String): Version of app

Response: Bash script text

### Search Resources
`GET /resources?query=<query>&categories=<categories>`

Search for learning resources.

If no search parameters are provided all resources will be returned.

Request:

- `query` (Optional, String): Natural language query
- `categories` (Optional, List[String]): Categories of learning resources

Response:

- `resource` (List[[Resource Model](#resource-model)])

## User Endpoints
`GET /users/login`

Login via OpenShift.

Request: None

Response:

- `authentication_token` (String): Use this to authenticate with the App API in
  the future

## Meta Endpoints
### Health Check
`GET /health`

Used to determine if server is operating fully.

Request: None

Response: None

# Deployment Script
A one line deployment command will be provided to users in the form:

```
curl -L https://api.kscout.io/apps/<app_id>/version/<version>/deploy.sh | bash
```

This script will allow users to tweak the values of `ConfigMap` and `Secret`
resources before applying them to a Kubernetes cluster.  

To facilitate this process the script will be automatically generated on the
server. It will contain a heredoc with the app's deployment JSON. For each of
the `ConfigMap` or `Secret` keys it will place a variable. It will prompt the
user for the value of this variable, or use the default value.
