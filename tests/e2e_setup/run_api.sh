#!/bin/bash

/usr/local/bin/containerd > ./daemon-local.log 2>&1 &
/usr/bin/dockerd --insecure-registry=0.0.0.0/0 >./daemon-local.log 2>&1 &
rm -rf /drone/test-reports/*
SWAGGER_CLIENT_PATH=/drone/harborclient/harbor_v2_swagger_client HARBOR_HOST=zdj-dev.local PYTHONPATH=/drone/tests/apitests/python python /drone/tests/apitests/python/api.py && junit-viewer --results=/drone/test-reports --save=/drone/test-reports/report.html