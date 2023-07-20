docker build -t val_node -f Dockerfile.val . --network host
docker build -t pr_node -f Dockerfile.peer . --network host
