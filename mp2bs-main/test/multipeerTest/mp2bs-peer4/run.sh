#!/bin/bash

docker build -t multipeer:latest .

docker container stop peer4 && docker container rm peer4

docker run -itd -p 192.168.10.3:4646:4646/udp --cap-add=NET_ADMIN --net n1 --name peer4 multipeer
docker network connect n2 peer4

docker exec -it peer4 sh
