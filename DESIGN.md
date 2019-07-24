# Design
API design.

# Table Of Contents
- [Overview](#overview)
- [Data Model](#data-model)
- [Endpoints](#endpoints)
  - [App Endpoints](#app-endpoints)
    - [Search Apps](#search-apps)
	- [Natural Search](#natural-search)
	- [Get App By ID](#get-app-by-id)
	- [App Pull Request Webhook](#app-pull-request-webhook)
	- [Search Tags](#search-tags)
	- [Search Categories](#search-categories)
	- [Get Deployment File](#get-deployment-file)
	- [Get Deployment Script](#get-deployment-script)
	- [Get Deployment Instructions](#get-deployment-instructions)
  - [Meta Endpoints](#meta-endpoints)
	- [Health Check](#health-check)
- [Deployment Script](#deployment-script)
- [Internal Metrics](#internal-metrics)

# Overview
HTTP RESTful API.  

Requests pass data via JSON encoded bodies except for in GET requests where data
will be passed via URL and query parameters.

Responses will always return JSON.

# Data Model
## App Model
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/models#App)  

Stored in the `apps` collection.  

# Endpoints
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers)  

Most endpoints do not require authentication.  

Those which do will be marked. Provide authentication as a bearer token in the
`Authorization` header.  

Endpoints which specify a response of `None` will return the 
JSON: `{"ok": true}`.

## App Endpoints
### Search Apps
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#AppSearchHandler)  

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
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#SmartSearchHandler)  

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
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#AppByIDHandler)  

`GET /apps/id/<app_id>`

Get application by ID.

Request:

- `app_id` (String)

Response:

- `app` ([App Model](#app-model))

### App Pull Request Webhook
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#WebhookHandler)  

`POST /apps/webhook`

GitHub will make a request to this endpoint every time a new pull request is 
made to submit an app.

Request:

- [GitHub Webhook Request](https://developer.github.com/webhooks/#payloads)
- [`PullRequestEvent`](https://developer.github.com/v3/activity/events/types/#pullrequestevent)

Response: None

### Search Tags
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#AppTagsHandler)  

`GET /apps/tags?query=<query>`

Get all available tags.

Request:

- `query` (Optional, String): Search string, if empty all tags will be returned

Response:

- `tags` (List[String])

### Search Categories
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#AppCategoriesHandler)  

`GET /apps/categories?query=<query>`

Get all available categories.

Request:

- `query` (Optional, String): Search string, if empty all categories will 
  be returned

Response:

- `categories` (List[String])

### Get Deployment File
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#AppsDeployResourcesHandler)  

`GET /apps/id/<app_id>/deployment.json`  

Get file with all an app's deployment resources.

Request:

- `app_id` (String): ID of app

Response: JSON text of deployment resources

### Get Deployment Script
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#AppsDeployHandler)  

`GET /apps/id/<app_id>/deploy.sh`

Get deployment script for version of app.  
See [deployment script](#deployment-script) for design details.

Request:

- `app_id` (String): ID of app

Response: Bash script text

### Get Deployment Instructions
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#DeployInstructionsHandler)  

`GET /apps/id/<app_id>/deployment-instructions`

Get instructions for how user should deploy application.

Request:

- `app_id` (String): ID of app

Response:

- `instructions` (String): Deploy instructions, contains newlines,
  markdown formatted

## Meta Endpoints
### Health Check
[Godoc](https://godoc.org/github.com/kscout/serverless-registry-api/handlers#HealthHandler)  

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

# Internal Metrics
The API server publishes internal Prometheus metrics.

Metrics spec:

> *Format*
> 
> - Namespace: `<namespace>`
>    - Subsystem: `<subsystem>`
>        - `<metric>`
>            - `<label>`
>
> ...

- Namespace: `serverless_registry_api`
  - Subsystem: `api`
    - `response_durations_milliseconds`
        - `path`
        - `method`
        - `status_code`
    - `handler_panics_total`
        - `path`
        - `method`
  - Subsystem: `jobs`
    - `run_durations_milliseconds`
        - `job_type`
        - `successful` 
