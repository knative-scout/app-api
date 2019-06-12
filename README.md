# Serverless Registry API
API which curates serverless applications.

# Table Of Contents
- [Overview](#overview)
- [Development](#development)
- [Deployment](#deployment)

# Overview
See [DESIGN.md](DESIGN.md)

# Development
The API server can be run locally.  

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
- `APP_DB_USER` (String): MongoDB user, defaults to `kscout-dev`
- `APP_DB_PASSWORD` (String): MongoDB password, defaults to `secretpassword`
- `APP_DB_NAME` (String): MongoDB database name, defaults
  to `kscout-serverless-registry-api-dev`
- `APP_GH_SECRET_KEY_PATH` (String): Path to GitHub App secret key
- `APP_GH_INTEGRATION_ID` (Integer): ID of GitHub APP, find in
  GitHub.com > Settings > Developer setting > GitHub Apps > YOUR GITHUB APP >
  General > About > App ID
- `APP_GH_INSTALLATION_ID` (Integer): Installation ID of GitHub APP, find in
  GitHub.com > Settings> Developer settings > GitHub Apps > YOUR GITHUB APP >
  Advanced > Recent Deliveries > CLICK ON ANY OF THE ITEMS > Request > Payload >
  `installation.id` field
- `APP_GH_REGISTRY_REPO_OWNER` (String): Owner of serverless application
  registry repository, defaults to `kscout`
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
  - *Pull requests*: Read-only
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

## Secrets
Deployment secrets must be set for a deployment.  

Create a JSON / YAML / TOML file with the following structure:

- `mongo` (Object): Secrets for Mongo database
  - `password` (String): Password to be used when creating account for API
- `github` (Object): GitHub secrets
  - `secretKeyPath` (String): Path to GitHub App secret key
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
oc rollout latest dc/ENV-serverless-registry-api
```

## Staging Deployment
The `deploy/staging.sh` script can be used to deploy one's local code to the 
staging environment.  

The staging environment is configured to be served under the 
`staging-api.kscout.io` domain.  

Try to coordinate with the team before using the staging environment to avoid 
stepping on each other's toes.

To deploy to the staging environment run:

```
./deploy/staging.sh rollout
```

To view the logs from the staging environment run:

```
./deploy/staging.sh logs
```
