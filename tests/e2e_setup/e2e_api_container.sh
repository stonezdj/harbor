#!/bin/bash

DOCKER_USER=<username>
DOCKER_PWD=<passwd>
HARBOR_HOST=zdj-dev.local
E2E_IMAGE="goharbor/harbor-e2e-engine:latest-api"
DIR=${PWD}
docker run -it --privileged -v $DIR/../../:/drone -v $DIR/../:/ca -v /etc/hosts:/etc/hosts -w /drone $E2E_IMAGE sh
