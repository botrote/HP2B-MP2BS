#!/bin/bash

docker build -t multipeer:latest .

docker container stop peer2 && docker container rm peer2

docker run -itd -p 192.168.10.3:4444:4444/udp --cap-add=NET_ADMIN --net n1 --name peer2 multipeer
docker network connect n2 peer2

docker exec -it peer2 sh
