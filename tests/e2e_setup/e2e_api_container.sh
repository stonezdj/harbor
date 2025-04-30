#!/bin/bash
DOCKER_USER=<username>
DOCKER_PWD=<passwd>
HARBOR_HOST=zdj-dev.local
E2E_IMAGE="goharbor/harbor-e2e-engine:latest-api"
docker run -i --privileged -v $DIR/../../:/drone -v $DIR/../:/ca -w /drone $E2E_IMAGE robot --exclude proxy_cache -v DOCKER_USER:${DOCKER_USER} -v DOCKER_PWD:${DOCKER_PWD} -v ip:${HARBOR_HOST}  -v ip1: -v http_get_ca:false -v HARBOR_PASSWORD:Harbor12345 /drone/tests/robot-cases/Group1-Nightly/Setup.robot /drone/tests/robot-cases/Group0-BAT/API_DB.robot