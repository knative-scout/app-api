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
exception of `APP_GH_TOKEN` and `APP_GH_WEBHOOK_SECRET`. 

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
- `APP_GH_WEBHOOK_SECRET` (String): Secret value which was provided during the
  [GitHub registry repository Webhook](#webhook) creation

## Run
Start the server by running:

```
go run .
```

# Deployment
Deployments are created for **environments**.  

An environment is a separated version of the deployment.  
Environments can be "production" or "staging".  

## GitHub
### Webhook
A webhook should exist for the
[app-repository](https://github.com/knative-scout/app-repository/settings/hooks/new).  
This webhook should send pull request events to the app pull request 
webhook endpoint.

For the "secret" value use a randomly generated string.  

### API Token
Generate an API token which has repository read access only.  

## Secrets
Deployment secrets must be set for a deployment.  

Create a JSON / YAML / TOML file with the following structure:

- `mongo` (Object): Secrets for Mongo database
  - `password` (String): Password to be used when creating account for API
- `github` (Object): GitHub secrets
  - `apiToken` (String): API token used to contact the GitHub API
  - `webhookSecret` (String): Secret used by GitHub to sign HMACs for requests
	made to the API webhook endpoint
  
## Deploy
Run the deploy script:

```
./deploy/deploy.sh -s SECRETS_FILE_FROM_ABOVE -e ENV
```

Where `ENV` can be a value like `prod` or `staging`.

Then trigger a rollout:


```
oc rollout latest dc/ENV-app-api
```
