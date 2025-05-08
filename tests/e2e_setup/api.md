# setup the api test env

## run the e2e_api_container.sh

```
cd tests/e2e_setup
./e2e_api_container.sh

```

## In the container console, run the following command

```
/usr/local/bin/containerd > ./daemon-local.log 2>&1 &
/usr/bin/dockerd --insecure-registry=0.0.0.0/0 >./daemon-local.log 2>&1 &
SWAGGER_CLIENT_PATH=/drone/harborclient/harbor_v2_swagger_client HARBOR_HOST=zdj-dev.local PYTHONPATH=/drone/tests/apitests/python python /app/api.py && junit-viewer --results=/drone/test-reports --save=/drone/test-reports/report.html
```

or just run 

```
bash -x /app/run_api.sh
```