#!/bin/bash


#setup environment
if ! sudo docker-compose -f ./docker/docker-compose.yml  up -d; then
  echo "Failed to setup local dynamo db"
  exit
fi

#run tests
go test ./...

sudo docker-compose -f ./docker/docker-compose.yml  down