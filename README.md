[![GoDoc](https://godoc.org/github.com/kscout/serverless-registry-api?status.svg)](https://godoc.org/github.com/kscout/serverless-registry-api)  

# Serverless Registry API
API which curates serverless applications.

# Table Of Contents
- [Overview](#overview)
- [Development](#development)
  - [Database](#database)
  - [Configuration](#development-configuration)
  - [Run](#run)
  - [Advanced Run](#advanced-run)
- [Deployment](#deployment)
  - [GitHub App](#github-app)
  - [Configuration](#deploy-configuration)
  - [Deploy](#deploy)
  - [Staging Deployment](#staging-deployment)
  - [GitHub Deployment Status](#github-deployment-status)

# Overview
See [DESIGN.md](DESIGN.md)

# Development
The API server can be run locally.  

A GitHub App is required to interact with the GitHub API.  
The KScout GitHub organization owns an app named "KScout Staging", use this for 
local development.

Follow the steps in the [Database](#database), [Configuration](#configuration),
and [Run](#run) sections.

## Database
Start a local MongoDB server by running:

```
make db
```

## Development Configuration
Configuration is passed via environment variables.  

Most configuration fields have default values which will work for local 
development. However a few fields must be set:

- `APP_BOT_API_SECRET` (String): Secret value used to authenticate with the 
  [bot API](https://github.com/kscout/bot-api)
- `APP_GH_INTEGRATION_ID` (Integer): ID of GitHub App
  - Find by going to: 
	[KScout Org. GitHub Apps](https://github.com/organizations/kscout/settings/apps) >
	YOUR GITHUB APP > General > About > App ID
- `APP_GH_INSTALLATION_ID` (Integer): Installation ID of GitHub APP
  - Find by going to:
	[KScout Org. GitHub Apps](https://github.com/organizations/kscout/settings/apps) >
	YOUR GITHUB APP > Advanced > Recent Deliveries > CLICK ON ANY OF THE ITEMS >
	Request > Payload > `installation.id` field
- `APP_GH_WEBHOOK_SECRET` (String): Secret value which was provided during the
  [GitHub App creation](#github-app)
  
You must also obtain the "KScout Staging" GitHub App private key. Send a message
to the Slack channel asking for this file. Then place it in the root of 
this repository.

You do not have to change any of the other configuration fields. Documentation 
for these fields follows:

- `APP_EXTERNAL_URL` (String): External URL from which HTTP server can
  be accessed. Should include any URL schemes, ports, paths, subdomains, ect.
  Should not include a trailing slash. Defaults to `http://localhost:5000`.
- `APP_SITE_URL` (String): URL of site, defaults to `https://kscout.io`
- `APP_BOT_API_URL` (String): URL of the bot API, defaults 
  to `https://bot.kscout.io`
- `APP_API_ADDR` (String): Address to bind API server, defaults to `:5000`
- `APP_METRICS_ADDR` (String): Address to bind metrics server, defaults 
  to `:9090`
- `APP_DB_HOST` (String): MongoDB host, defaults to `localhost`
- `APP_DB_PORT` (Integer): MongoDB port, defaults to `27017`
- `APP_DB_USER` (String): MongoDB user, defaults to `kscout-dev`
- `APP_DB_PASSWORD` (String): MongoDB password, defaults to `secretpassword`
- `APP_DB_NAME` (String): MongoDB database name, defaults
  to `kscout-serverless-registry-api-dev`
- `APP_GH_PRIVATE_KEY_PATH` (String): Path to GitHub App's private key
- `APP_GH_REGISTRY_REPO_OWNER` (String): Owner of serverless application
  registry repository, defaults to `kscout`
- `APP_GH_REGISTRY_REPO_NAME` (String): Name of serverless application
  registry repository, defaults to `serverless-apps`

## Run
Start the server by running:

```
go run .
```

## Advanced Run
### Update Apps
Force the server to rebuild its database state by passing the 
`-update-apps` flag:

```
go run . -update-apps
```

To have the server make a request to the bot API's new apps endpoint after it is
done updating apps pass the `-notify-bot-api` flag as well:

```
go run . -update-apps -notify-bot-api
```

This makes the server import data from the serverless registry repository.

### Seed Data
Insert seed data into the database by passing the `-seed` flag:

```
go run . -seed
```

This will load the JSON files in the `./seed-data` directory into the database.

### Validate Registry Repository Pull Request
To run a validation job for a pull request in the [serverless application 
registry repository](https://github.com/kscout/serverless-apps) pass the 
`-validate-pr PR_NUM` flag:

```
go run . -validate-pr PR_NUM
```

This will ensure the applications modified by the PR are correctly formatted.  
The job will set a check run status and make a comment on the PR based on the
results of the format validation.

### Mock Webhook Request
To make a mock webhook request to the 
[app pull request webhook](DESIGN.md#app-pull-request-webhook) pass the 
`-mock-webhook REQ_BODY_FILE` and `-mock-webhook-event EVENT_NAME` flags:

```
go run . -mock-webhook REQ_BODY_FILE -mock-webhook-event EVENT_NAME
```

`REQ_BODY_FILE` should be the name of a JSON file which will be sent as the 
request body.  
`EVENT_NAME` should be the name of a GitHub event which will be sent as the
`X-Github-Event` header value.

This will make an HTTP POST request to the server specified by the 
`APP_EXTERNAL_URL` configuration environment variable.  
This request will contain the `X-Hub-Signature` header which is a signed 
HMAC of the request body. The `APP_GITHUB_WEBHOOK_SECRET` key will be used
to sign this HMAC.

# Deployment
To deploy:

1. [Ensure a GitHub App exists](#github-app)
2. [Configure](#deploy-configuration)
3. [Deploy](#deploy)

## GitHub App
### Create
Create a GitHub App with the following parameters:

- **Name**: `KScout`
- **Description**: `Smart App Hub for Serverless Knative Apps by Red Hat.`
- **Homepage URL**: `https://kscout.io`
- **User authorization callback URL**: `https://api.kscout.io/auth/github_app/callback`
- **Webhook URL**: `https://api.kscout.io/apps/webhook`
- **Webhook secret**: A secret random string
- **Permissions**:
  - *Checks*: Read & write
  - *Repository contents*: Read-only
  - *Pull requests*: Read & write
- **Subscribe to events**:
  - *Check run*
  - *Pull request*

### Set Logo
Once created set the logo to 
[`logo.png` from the meta repository](https://github.com/kscout/meta/blob/master/logo.png).

### Generate Private Key
Go to the "private keys" section of the GitHub App settings page and 
generate a private key.

### Install
Navigate to the "Install App" tab in the left menu. Click the "Install" button 
for the `kscout` organization.  

On the next page select "Only select repositories" and 
choose `kscout/serverless-apps`.

Click "Install".

## Deploy Configuration
Create a copy of `deploy/values.secrets.example.yaml` named 
`deploy/values.secrets.ENV.yaml` for whichever deployment environment you wish
to configure.

Edit this file with your own values.

Never commit this file.

## Deploy
Initialize submodules:

```
git submodule update --init --recursive
```

Deploy to production:

```
make deploy-prod
```

If this is the first time production has been deployed run:

```
make rollout-prod
```

The `master` branch will automatically be deployed to the `prod` environment.  

## Staging Deployment
Local code can be deployed to the staging environment.  

Spin up the staging environment if it doesn't exist already:

```
make deploy-staging
make rollout-staging
```

Build local code and deploy to environment:

```
make push
```

## GitHub Deployment Status
GitHub has 
[a page](https://github.com/kscout/serverless-registry-api/deployments) which 
tracks the commit currently deployed to an environment.

To create a status for a deployment run:

```
make gh-deploy
```

You can optionally set the `STATE` and `REF` variable. See the target in the 
`Makefile` for more details.
