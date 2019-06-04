# App API
API which manages applications.

# Table Of Contents
- [Overview](#overview)
- [Development](#development)
- [Deployment](#deployment)

# Overview
See [DESIGN.md](DESIGN.md)

# Development
## Database
Start a local MongoDB server by running:

```
make db
```

## Configuration
The default values will work for local development.  

Configuration is passed via environment variables:

- `APP_HTTP_ADDR` (String): Address to bind server, defaults to `:5000`
- `APP_DB_HOST` (String): MongoDB host, defaults to `localhost`
- `APP_DB_PORT` (Integer): MongoDB port, defaults to `27017`
- `APP_DB_USER` (String): MongoDB user, defaults to `knative-scout-dev`
- `APP_DB_PASSWORD` (String): MongoDB password, defaults to `secretpassword`
- `APP_DB_NAME` (String): MongoDB database name, defaults
  to `knative-scout-app-api-dev`
- `APP_GH_TOKEN` (String): GitHub API token with repository read permissions

## Run
Start the server by running:

```
go run .
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

# Deployment
## Kubernetes
Set the Mongo DB database password:

```
echo "PUT YOUR PASSWORD HERE" | base64 | tee password
kubectl -n knative-scout create secret generic mongo-credentials --from-file password
rm password
```

Deploy the API server and Mongo:

```
make deploy
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
