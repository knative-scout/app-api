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
## User Model
`users` collection.

Schema:

- `username` (String)
- `profile_picture` (String): Binary image data
- `cluster` (Object)
  - `name`: Name of OpenShift cluster
  
## App Model
`apps` collection.

Schema:

- `app_id` (String)
- `name` (String)
- `tagline` (String)
- `description` (String)
- `screenshot_urls` (List[String])
- `logo_url` (String)
- `tags` (List[String])
- `verification_status` (String): One of `pending`, `verifying`, `good`, `bad`
- `github_link` (String)
- `version` (String)
- `author` (String)
- `maintainer` (String)

# Endpoints
Most endpoints to not require authentication.  

Those which do will be marked. Provide authentication as a bearer token in the
`Authorization` header.  

## App Endpoints
### Search Apps
`GET /apps?query=<query>&tags=<tags>&categories=<categories>`

Search serverless apps in hub.

If no search parameters are provided all applications will be returned.

Request:

- `query` (Optional, String): Natural language query
- `tags` (Optional, List[String]): Tags applications must have
- `categories` (Optional, List[String]): Categories applications must be part of

Response:

- `apps` (List[[App Model](#app-model)])

## User Endpoints
`GET /users/login`

Login via OpenShift.

Request: None

Response:

- `authentication_token` (String): Use this to authenticate with the App API in
  the future

## Cluster Endpoints
### List Clusters
`GET /clusters`

Lists available OpenShift clusters.

Authentication required.

Request: None

Response:

- `clusters` (List[Object]): Objects with keys
  - `id` (String)
  - `name` (String)

### Deploy To Cluster
`POST /clusters/<id>/deploy?app_id=<app_id>`

Deploy application to OpenShift cluster.

Request:

- `id` (String): ID of OpenShift cluster
- `app_id` (String): ID of app to deploy

Response: None

### Get Deploy Instructions
`GET /clusters/deploy_instructions?app_id=<app_id>`

Get manual deploy instructions for app.

Request:

- `app_id` (String): ID of app to return instructions for

Response:

- `instructions` (String)
