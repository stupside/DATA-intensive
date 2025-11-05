# Assignment 4 - Client & Server

This assignment contains a client application for device onboarding and file sharing.

## Server

First generate a certificate for the server.
```bash
chmod +x ./scripts/generate-certs.sh

./scripts/generate-certs.sh
```

The server uses PostgreSQL for metadata and MongoDB for streaming state.
Ensure both databases are running and configured in **config.yml**.

With docker run the following command to set them up. 
```bash
docker compose up -d
```

The configuration for the server can be found in **config.yml** please modify if needed.

When the server is properly configured, you can run it.
```bash
go run cmd/server/main.go
```

## Client

An example is available in the [example folder](example/README.md)