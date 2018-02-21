#!/bin/bash

BUILD_IMAGE='amitsaha/aws_asg_lifecycle_consumer-deb'
FPM_IMAGE='amitsaha/aws_asg_lifecycle_consumer-deb-fpm'

docker build -t $BUILD_IMAGE -f Dockerfile-go1.8 .
containerID=$(docker run --detach $BUILD_IMAGE)
docker cp $containerID:/aws_asg_lifecycle_consumer .
sleep 1
docker rm $containerID

# Package it up
version=`git rev-parse --short HEAD`
VERSION_STRING="$(cat VERSION)-${version}"

docker build --build-arg \
    version_string=$VERSION_STRING \
    -t $FPM_IMAGE -f Dockerfile-fpm .
containerID=$(docker run -dt $FPM_IMAGE)
# docker cp does not support wildcard:
# https://github.com/moby/moby/issues/7710
mkdir dpkg-source

docker cp $containerID:/aws_asg_lifecycle_consumer-package/aws-asg-lifecycle-consumer-${VERSION_STRING}.deb dpkg-source/.
sleep 1
docker rm -f $containerID
rm aws_asg_lifecycle_consumer
