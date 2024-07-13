#!/bin/bash

# Navigate to the project directory
cd /home/ec2-user/ethglobal-2o24 || (echo "error" & exit)

# Pull the latest changes from the repository
git pull origin master

# Build the Docker image
sudo docker build -t go-rest-api .

# Stop and remove the existing container if it exists
sudo docker stop go-rest-api || true
sudo docker rm go-rest-api || true

# Run the new Docker container
sudo docker run -d --name go-rest-api -p 80:8080 go-rest-api

