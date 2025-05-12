#!/bin/bash

# HARBOR_HOST=zdj-dev.local
E2E_IMAGE="firstfloor/harbor-e2e-engine:api-slim"
DIR=${PWD}
docker run -it --privileged -v $DIR/../../:/drone -v $DIR/../:/ca -v /etc/hosts:/etc/hosts -e https_proxy=http://${https_proxy} -e http_proxy=http://${http_proxy}  -w /drone $E2E_IMAGE sh
