FROM ubuntu:20.04

RUN rm -rf /var/lib/apt/lists/*

RUN apt-get update
RUN apt-get install -y gcc net-tools
RUN apt-get install -y iproute2

WORKDIR /mp2bs

COPY peers peers
COPY peer.sh peer.sh
COPY config config
