#!/bin/bash

# HARBOR_HOST=zdj-dev.local
E2E_IMAGE="firstfloor/harbor-e2e-engine:api-slim"
DIR=${PWD}
wget --no-check-certificate https://${HARBOR_HOST}/api/v2.0/systeminfo/getcert -O ca.crt
docker run -it --privileged -v $DIR/../../:/drone -v $DIR/../:/ca -v -v ${PWD}/ca.crt:/etc/docker/certs.d/${HARBOR_HOST}/ca.crt -v /etc/hosts:/etc/hosts -e https_proxy=http://${https_proxy} -e http_proxy=http://${http_proxy}  -w /drone $E2E_IMAGE sh
