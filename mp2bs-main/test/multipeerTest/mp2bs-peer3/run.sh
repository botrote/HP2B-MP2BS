#!/bin/bash

docker build -t multipeer:latest .

docker container stop peer3 && docker container rm peer3

docker run -itd -p 192.168.10.3:4545:4545/udp --cap-add=NET_ADMIN --net n1 --name peer3 multipeer
docker network connect n2 peer3

docker exec -it peer3 sh
