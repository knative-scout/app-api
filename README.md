# App API
API which manages applications.

# Table Of Contents
- [Overview](#overview)
- [Development](#development)
- [Deployment](#deployment)

# Overview
See [DESIGN.md](DESIGN.md)

# Development
The App API server can be run locally.  

Follow the steps in the [Database](#database), [Configuration](#configuration),
and [Run](#run) sections.

## Database
Start a local MongoDB server by running:

```
make db
```

## Configuration
For local development the default configuration values will suffice with the 
exception `APP_GH_TOKEN`.

Configuration is passed via environment variables.

- `APP_HTTP_ADDR` (String): Address to bind server, defaults to `:5000`
- `APP_DB_HOST` (String): MongoDB host, defaults to `localhost`
- `APP_DB_PORT` (Integer): MongoDB port, defaults to `27017`
- `APP_DB_USER` (String): MongoDB user, defaults to `knative-scout-dev`
- `APP_DB_PASSWORD` (String): MongoDB password, defaults to `secretpassword`
- `APP_DB_NAME` (String): MongoDB database name, defaults
  to `knative-scout-app-api-dev`
- `APP_GH_TOKEN` (String): GitHub API token with repository read permissions
- `APP_GH_REGISTRY_REPO_OWNER` (String): Owner of serverless application
  registry repository, defaults to `knative-scout`
- `APP_GH_REGISTRY_REPO_NAME` (String): Name of serverless application
  registry repository, defaults to `serverless-apps`

## Run
Start the server by running:

```
go run .
```

# Deployment
## Kubernetes
1. Set secrets
  - Create copy of `deploy/secrets.ex.yaml` named `deploy/secrets.yaml`
  - Replace the placeholder values with the correct base64 encoded values
2. Deploy
   - Deploy the database:
     ```
	 ./deploy/deploy.sh up db
	 ```
   - Deploy the app API server:
     ```
	 ./deploy/deploy.sh up app
	 ```

## Temporary Open Shift
The `tmpk` script wraps `kubectl` with the required arguments to connect to the
48 hour Open Shift clusters.

Set the `TMPK_TOKEN` and `TMPK_N` environment variables. See the `tmpk` file 
for details about what to set these environment variables to.

Use the `tmpk` script as if it was `kubectl`:

```
./tmpk get all
```

## GitHub
### Webhook
A webhook should exist for the
[app-repository](https://github.com/knative-scout/app-repository/settings/hooks/new).  
This webhook should send pull request events to the app pull request 
webhook endpoint.

### API Token
Generate an API token which has repository read access only.  
Provide to application via `APP_GH_TOKEN` environment variable.
