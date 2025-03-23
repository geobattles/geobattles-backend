# GeoBattles

GeoBattles is an open-source interactive geolocation guessing game where players compete to identify locations from Google Street View panoramas.

## 🌍 About the Game

GeoBattles features a **Battle Royale** mode where players can play solo or compete against others to guess locations, with limited lives.

The game offers real-time multiplayer functionality, live statistics, and interactive map elements.

#### This repository contains code for the backend server written in go. Frontend can be found [here](https://github.com/geobattles/geobattles-frontend).

## 🚀 Getting Started
## Production Setup
The easiest way to run both the frontend and backend is using the prebuilt Docker images using Docker Compose.

<details>
<summary>📝 Google API Setup (required)</summary>

This project relies on Google Maps and StreetView APIs. Head over to Google Cloud [console](https://console.cloud.google.com), enable `Maps JavaScript API` and `Street View Static API` and generate an API key.

> :warning:  
> Be sure to properly secure your API key and setup usage quotas to avoid unexpected charges. You can use different keys for frontend and backend and further restrict them. Frontend only needs `Maps JavaScript API` and can be restricted to your website domain. Backend only needs `Street View Static API` and can be restricted to your server's public IP.

</details><br/>

1. Create a folder for the project, for example `geobattles`.
2. Create a `docker-compose.yaml` file with the following content:
```docker
services:
  backend:
    image: ghcr.io/geobattles/geobattles-backend:latest
    restart: unless-stopped
    environment:
      GMAPS_API: "<GMAPS_API_KEY>"
      DB_HOST: "database"
      DB_USER: "geobattles"
      DB_PASSWORD: "<POSTGRES_PWD>"
      LOG_LEVEL: "INFO"
    ports:
      - "8080:8080"
    depends_on:
    - database

  frontend:
    image: ghcr.io/geobattles/geobattles-frontend:latest
    restart: unless-stopped
    environment:
      NUXT_PUBLIC_GMAPS_API: "GMAPS_API_KEY"
      NUXT_PUBLIC_BACKEND_HOST: "<PUBLIC_BACKEND_URL>"
    ports:
      - "3000:3000"
    depends_on:
    - backend

  database:
    image: postgres:17-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: "geobattles"
      POSTGRES_PASSWORD: "<POSTGRES_PWD>"
      POSTGRES_DB: "db"
    volumes:
      - .db:/var/lib/postgresql/data
```
3. Run it with `docker compose up -d`

### SSL Configuration
You will also want to setup SSL termination. Below are example nginx configurations for use with the [swag](https://github.com/linuxserver/docker-swag) container.

<details>
<summary>nginx server block</summary>

```nginx
# backend
server {
    listen 443 ssl;
    listen [::]:443 ssl;

    server_name <api.yourdomain.tld>;

    include /config/nginx/ssl.conf;

    client_max_body_size 0;

    location / {
        include /config/nginx/proxy.conf;

        proxy_pass http://<backend_local_ip>:8080;
    }
}

# frontend
server {
    listen 443 ssl;
    listen [::]:443 ssl;

    server_name <yourdomain.tld>;

    include /config/nginx/ssl.conf;

    client_max_body_size 0;

    location / {
        include /config/nginx/proxy.conf;

        proxy_pass http://<frontend_local_ip>:3000;
    }
}
```
</details><br/>

## 🛠️ Development setup
### Requirements

-   Go >=1.24 or Docker
-   API key for Google Street View Static API
-   Postgres database

### Setup Steps
1. Clone the repository
```
git clone https://github.com/geobattles/geobattles-backend
cd geobattles-backend
```

2. Chose your development method
#### With Go installed locally
```bash
# Copy environment file and configure it
cp example.env to .env

# Download dependencies and run:
go get .
go run .
```

#### With Docker
```shell
# Build image
docker build  . -t geobattles-backend

# Run container
docker run \
  -p 8080:8080 \
  -e GMAPS_API=<GMAPS_API_KEY> \
  -e DB_HOST=<POSTGRES_URL> \
  -e DB_PASSWORD=<POSTGRES_PWD> \
  geobattles-backend
```

> [!NOTE]  
> For all available env variables see `example.env`
