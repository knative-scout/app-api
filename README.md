# Serverless Registry API
API which curates serverless applications.

# Table Of Contents
- [Overview](#overview)
- [Development](#development)
  - [Database](#database)
  - [Configuration](#Development configuration)
  - [Run](#run)
  - [Advanced Run](#advanced-run)
- [Deployment](#deployment)
  - [GitHub App](#github-app)
  - [deploy-Configuration](#deploy-configuration)
  - [Deploy](#deploy)
  - [Staging Deployment](#staging-deployment)

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

## Configuration
Configuration is passed via environment variables.  

Most configuration fields have default values which will work for local 
development. However a few fields must be set:

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

- `APP_HTTP_ADDR` (String): Address to bind server, defaults to `:5000`
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

# Deployment
Deployments are created for **environments**.  

An environment is a self contained deployment. Different environments hold code 
with varying levels of stability.  

The production (or "prod") environment holds the most stable code.  
The staging environment can hold less stable code, or code who's stability is 
not yet known.

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
The following environment variables must be set (see
[Development#Configuration](#development-configuration) for documentation of
these variables):

- `APP_DB_PASSWORD`
- `APP_GH_INTEGRATION_ID`
- `APP_GH_INSTALLATION_ID`
- `APP_GH_WEBHOOK_SECRET`
- `APP_GH_PRIVATE_KEY_PATH`

The indicated by `APP_GH_PRIVATE_KEY` must also contain the appropriate GitHub 
application's private key.

## Deploy
Run the deploy script:

```
./deploy/deploy.sh -r
```

## Staging Deployment
The `deploy/deploy.sh` script can also be used to deploy one's local code to
the staging environment.  

The staging environment is configured to be served under the 
`staging-api.kscout.io` domain.  

Try to coordinate with the team before using the staging environment to avoid 
stepping on each other's toes.

To deploy to the staging environment run:

```
./deploy/deploy.sh -e staging -rb
```

