#!/bin/bash

set -e

TAG=${IMAGE_TAG:-'latest'}

docker build --build-arg ARGO_VERSION="${TAG}" -t "${IMAGE_NAMESPACE:-$(whoami)}/athena-ui:${TAG}" .

if [ "$DOCKER_PUSH" == "true" ]
then
    docker push "${IMAGE_NAMESPACE:-$(whoami)}/athena-ui:${TAG}"
fi
