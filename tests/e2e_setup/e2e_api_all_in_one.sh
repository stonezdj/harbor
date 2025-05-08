#!/bin/bash
set -e
E2E_IMAGE="firstfloor/harbor-e2e-engine:api-slim"
DIR=${PWD}
docker run -i --privileged -v $DIR/../../:/drone -v $DIR/../:/ca -v /etc/hosts:/etc/hosts -e HARBOR_HOST=${HARBOR_HOST} -w /drone ${E2E_IMAGE} bash -x /app/run_api.sh