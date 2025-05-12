#!/bin/bash

/usr/local/bin/containerd > ./daemon-local.log 2>&1 &
/usr/bin/dockerd --insecure-registry=0.0.0.0/0 >./daemon-local.log 2>&1 &
rm -rf /drone/test-reports/*
cp /app/Makefile .
cp /app/openapi-generator-cli.jar .
make swagger_client
SWAGGER_CLIENT_PATH=/drone/harborclient/harbor_v2_swagger_client PYTHONPATH=/drone/tests/apitests/python python /app/api.py 
exit_code=$?
junit-viewer --results=/drone/test-reports --save=/drone/test-reports/report.html
exit $exit_code