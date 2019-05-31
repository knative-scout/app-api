# App API
API which manages applications.

# Table Of Contents
- [Overview](#overview)
- [Development](#development)
- [Deployment](#deployment)

# Overview
Design TBD.

# Development
Written in Go using the native HTTP library.  

Start a local MongoDB server by running:

```
make db
```

Start the server by running:

```
go run .
```

# Deployment
Configuration is passed via environment variables:

- `APP_HTTP_ADDR` (String): Address to bind server, defaults to `:5000`
- `APP_DB_CONN_URL` (String): MongoDB connection URI, defaults to connecting to
  the local database started by `make db`
