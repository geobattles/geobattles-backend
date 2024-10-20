# Server backend for Geoguessr Clone written in Go

# Requirements

-   Go or Docker
-   API key for Google Street View Static API
-   Postgres database

# Running



## Setup postgres database
```
docker run -d -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgrespwd -e POSTGRES_DB=db -p 5432:5432 postgres
```

_Configure environment variables or add them to .env file. See example.env_

```
git clone https://github.com/slinarji/go-geo-server
cd go-geo-server
go get .
go run .
```

# Docker

```
docker build  . -t <image name>
docker run -p 8080:8080 -e GMAPS_API=<your api key> <image name>
```
