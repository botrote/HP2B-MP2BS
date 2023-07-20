#!/bin/bash

docker build -t multipeer:latest .

docker container stop peer0 && docker container rm peer0

docker run -itd -p 192.168.10.3:4242:4242/udp --cap-add=NET_ADMIN --net n1 --name peer0 multipeer
docker network connect n2 peer0

docker exec -it peer0 sh
