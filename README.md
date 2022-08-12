# Server backend for Geoguessr Clone written in Go

# Requirements

-   Go or Docker
-   API key for Google Street View Static API

# Running

_export your api key to GMAPS_API env or put it into .env file_

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
