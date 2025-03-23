# GeoBattles

GeoBattles is an open-source interactive geolocation guessing game where players compete to identify locations from Google Street View panoramas.

## 🌍 About the Game

GeoBattles features a **Battle Royale** mode where players can play solo or compete against others to guess locations, with limited lives.

The game offers real-time multiplayer functionality, live statistics, and interactive map elements.

## This repository contains code for the backend server written in go. Frontend can be found [here](https://github.com/geobattles/geobattles-frontend).

# Production setup
The easiest way to run both the frontend and backend is using the prebuild docker images using docker compose. Create a folder for your setup, for example `geobattles`. Copy and paste the following into a file named `docker-compose.yaml` inside that folder. Run it with `docker compose up -d`
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

You will also want to run a reverse proxy in front of the services for ssl termination. The below nginx snippets work with linuxserver [swag](https://github.com/linuxserver/docker-swag) container.
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



# Development setup

-   Go >=1.24 or Docker
-   API key for Google Street View Static API
-   Postgres database

### Clone the repo
```
git clone https://github.com/geobattles/geobattles-backend
cd geobattles-backend
```

### Run server either locally or with docker
#### Locall go install
Copy `example.env` to `.env` and insert your values.

Download dependencies and run:
```
go get .
go run .
```
### With Docker
Build and run Docker image:

`docker build  . -t <image_name>`

```
docker run \
-p 8080:8080 \
-e GMAPS_API=<GMAPS_API_KEY>
-e DB_HOST=<POSTGRES_URL>
-e DB_PASSWORD=<POSTGRES_PWD>
<image_name>
```

For all available env variables see `example.env`
