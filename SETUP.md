# Setup

## Prerequisites

- Docker Desktop with Docker Compose
- Node.js and npm to run the frontend locally

## Start the database and backend

From the project root:

```powershell
docker compose up --build -d
docker compose ps
docker compose logs -f backend
```

Compose starts PostGIS on host port `5433`, creates the database schema, builds
the Go backend image, and exposes the API on port `8080`. The database data is
stored in the `pgdata` volume.

If host port `8080` is already in use, choose a different host port for the API
before starting Compose:

```powershell
$env:API_PORT = "8081"
docker compose up --build -d
```

In that case, use `http://localhost:8081` for the API and frontend
`VITE_API_URL`.

The migration in `backend/migrations/001_init.sql` is automatically applied
when PostgreSQL creates a new, empty data directory. If `pgdata` already
exists, apply the migration explicitly:

```powershell
docker compose exec db psql -U geofence -d geofence_db -f /docker-entrypoint-initdb.d/001_init.sql
```

Check the API and database connection:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

The response's `status` should be `ok`. The backend connects to the database
through Compose's internal hostname `db:5432`; applications running on Windows
can connect to PostgreSQL at `localhost:5433`.

## API smoke test

Check the vehicles endpoint:

```powershell
Invoke-RestMethod http://localhost:8080/vehicles
```

Create a vehicle:

```powershell
$body = @{
  vehicle_number = "KA-01-AB-1234"
  driver_name = "Example Driver"
  vehicle_type = "truck"
  phone = "+1234567890"
} | ConvertTo-Json

Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/vehicles `
  -ContentType "application/json" `
  -Body $body
```

Other available routes include:

- `GET, POST /geofences`
- `POST /vehicles/location`
- `GET /vehicles/location/{vehicle_id}`
- `GET, POST /alerts` (alert configuration uses `POST /alerts/configure`)
- `GET /violations/history`
- WebSocket: `ws://localhost:8080/ws/alerts`

## Run the frontend locally

In another PowerShell window:

```powershell
Set-Location .\frontend
npm ci
npm run dev
```

The frontend's default API URL is `http://localhost:8080`. To use another
backend URL, set `VITE_API_URL` before starting Vite.

## Stop the services

From the project root:

```powershell
docker compose down
```

This stops the containers while preserving the database volume. Avoid
`docker compose down -v` unless you intend to delete the database data.

## Build and publish the backend image

The published image is `jsharini/geofence-backend`. To build and publish a new
version after making backend changes:

```powershell
docker login
docker build -t jsharini/geofence-backend:1.1 .\backend
docker push jsharini/geofence-backend:1.1
```

## Deploy the backend to Render

The backend image is published at
`docker.io/jsharini/geofence-backend:1.1`. To deploy it:

1. In Render, create a PostgreSQL database in the same region as the web
   service. Render Postgres supports PostGIS.
2. Connect to the database using the PSQL command shown on its Render page and
   execute the SQL in `backend/migrations/001_init.sql` to create the schema.
3. Create a **Web Service** using **Existing Image** and enter
   `docker.io/jsharini/geofence-backend:1.1`.
   If the Docker Hub repository is private, add a registry credential in
   Render's workspace settings.
4. Configure the web service to use port `10000` (Render's default `PORT`), and
   set `DATABASE_URL` to the database's **Internal Database URL**. Keep the
   database URL private; do not commit it to source control.
5. Set the health check path to `/health`, deploy, and verify that the
   endpoint returns `status: ok`.

The server reads Render's `PORT` environment variable and falls back to `8080`
for local use. The Compose hostname `db` is only valid inside the local Compose
network. Do not use the example database password in production.

## Architecture

- `backend/`: Go HTTP API, WebSocket alert hub, and PostgreSQL migration
- `frontend/`: React/Vite dashboard
- `docker-compose.yml`: local PostGIS database and Go backend services
