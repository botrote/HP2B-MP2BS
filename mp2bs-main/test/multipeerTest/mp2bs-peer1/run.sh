#!/bin/bash

docker build -t multipeer:latest .

docker container stop peer1 && docker container rm peer1

docker run -itd -p 192.168.10.3:4343:4343/udp --cap-add=NET_ADMIN --net n1 --name peer1 multipeer
docker network connect n2 peer1

docker exec -it peer1 sh
