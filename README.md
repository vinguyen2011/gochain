## Start docker container for fabric and peers

cd to this directory
docker-compose -f docker-compose-gettingstarted.yml build
docker-compose -f docker-compose-gettingstarted.yml up -d

## Start proxy server

You should wait a bit for all the container and peer up and running
GOPATH=$PWD node deploy.js

## Deploy

You can use postman or other similar tool
POST localhost:3000/deploy
body:
{
	"data": [
         "a",
         "100"
      ]
}

## Invoke
POST localhost:3000/invoke
body:
{
	"data": [
          "move",
         "a",
         "b",
         "400"
      ]
}

## Query a state
POST localhost:3000/query
body:
{
	"data": [
          "query",
         "a"
      ]
}


## Query all states within a range
POST localhost:3000/query
body:
{
	"data": [
          "queryAll",
         "a",
         "z"
      ]
}