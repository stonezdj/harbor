#!/bin/bash

HARBOR_SRC_FOLDER=$(realpath ../../)
echo ${HARBOR_SRC_FOLDER}

testcase_tag=$1

# If the testbed network type is private need to set NETWORK_TYPE private, default is public
if [ -z "${testcase_tag}" ]; then
    docker run -it --privileged -v /var/log/harbor:/var/log/harbor -v /etc/hosts:/etc/hosts -v ${HARBOR_SRC_FOLDER}:/drone -v ${HARBOR_SRC_FOLDER}/tests/harbor_ca.crt:/ca/ca.crt -v /dev/shm:/dev/shm -e NETWORK_TYPE=public -e PIP_INDEX_URL=https://packages.vcfd.broadcom.net/artifactory/api/pypi/pypi-virtual/simple -w /drone registry.goharbor.io/harbor-ci/goharbor/harbor-e2e-engine:latest-ui /bin/bash
else
    docker run  --privileged -v /var/log/harbor:/var/log/harbor -v /etc/hosts:/etc/hosts -v ${HARBOR_SRC_FOLDER}:/drone -v ${HARBOR_SRC_FOLDER}/tests/harbor_ca.crt:/ca/ca.crt -v /dev/shm:/dev/shm -e NETWORK_TYPE=public -e PIP_INDEX_URL=https://packages.vcfd.broadcom.net/artifactory/api/pypi/pypi-virtual/simple -w /drone registry.goharbor.io/harbor-ci/goharbor/harbor-e2e-engine:latest-ui /drone/tests/e2e_setup/single_test.sh ${testcase_tag}
fi

