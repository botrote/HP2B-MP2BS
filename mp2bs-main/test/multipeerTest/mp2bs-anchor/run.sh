#!/bin/bash

docker build -t multipeer:latest .

docker container stop anchor && docker container rm anchor

docker run -itd -p 192.168.10.3:4251:4251/udp --cap-add=NET_ADMIN --net n1 --name anchor multipeer
docker network connect n2 anchor

docker exec -it anchor sh
