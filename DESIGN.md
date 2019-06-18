# Design
API design.

# Table Of Contents
- [Overview](#overview)
- [Data Model](#data-model)
- [Endpoints](#endpoints)

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
- `deployment_file_urls` (List[String])
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
`GET /apps/<app_id>/deployment.yaml`  

Get file with all an app's deployment resources.

Request:

- `app_id` (String): ID of app

Response: YAML text of deployment resources

### Get Deployment Parameters
`GET /apps/<app_id>/parameters`

Get parameters of an app's deployment.  

Parameters are `ConfigMap` or `Secret` resource keys which can be customized by
the user if they want.  

The results of this endpoint are meant to be parsed by a bash script. To make 
this possible data is grouped into entries. Each entry has multiple fields.  
Entries are separated by newlines. Fields are separated by spaces.

Request:

- `app_id` (String): ID of app

Response:

Newline separated list of entries. Each entry has 4 fields: 

- `ID`: Unique name for parameter. This string is guaranteed to appear exactly 
  once in the parameterized version of the deployment file. It will be located
  in the location where the value of the parameter should be placed
- `KEY`: Name of parameter
- `DEFAULT`: Base64 encoded default value of parameter
- `BASE64`: Indicates if the value of this parameter must be base64 encoded 
  before being placed in the parameterized deployment file.
  
Entries are placed in the following order on a line:

```
ID KEY DEFAULT BASE64
```
  
### Get Parameterized Deployment File
`GET /apps/<app_id>/parameterized-deployment.yaml`  

Get file with all an app's deployment resources.  
Any keys in `ConfigMap` or `Secret` resources will have placeholder values.  

These placeholder values are the `ID` fields from the 
[Get Deployment Parameters](#get-deployment-parameters) endpoint. Each will be
unique to the entire file and can be safely found-and-replaced with the actual
parameter value.

Request:

- `app_id` (String): ID of app

Response: YAML text of deployment resources

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
