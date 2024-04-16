#!/bin/bash

if [ "$#" == "1" ]
then
    version=$1
else
    version="master"
fi

docker pull debian:bookworm
docker build -t gofaxip_build .
docker run --rm -v $PWD/..:/input -v $PWD:/output -e VERSION=$version gofaxip_build
