# Python API tests

These scripts exercise a **running** Harbor instance via the generated OpenAPI client (`v2_swagger_client`) and helpers under [`library/`](library/). Run them from the **Harbor repository root** so paths in [`testutils.py`](testutils.py) resolve correctly.

## Prerequisites

- **Python 3.10+**
- **Java** — used by `make swagger_client` with OpenAPI Generator
- **Network access** — `make swagger_client` downloads the OpenAPI Generator JAR; the client is generated from `api/v2.0/swagger.yaml` in this repository

## 1. Generate and install the Swagger client

From the repository root:

```bash
make swagger_client
```

This creates `harborclient/harbor_v2_swagger_client`, runs `pip install` there (installs `v2_swagger_client`), and installs the `docker` Python package.

## 2. Environment variables

| Variable | Purpose |
|----------|---------|
| `HARBOR_HOST` | Hostname or IP of Harbor (**no** `https://`); required for most tests |
| `SWAGGER_CLIENT_PATH` | If imports fail, set to `./harborclient` (repository root relative) |
| `HARBOR_HOST_SCHEMA` | API URL scheme; default `https` |
| `DOCKER_USER` / `DOCKER_PWD` | Registry credentials for tests that push or pull images |
| `TEARDOWN` | Cleanup of created resources; default `true` (`true`/`yes` enable teardown) |

Defaults for API login in `testutils` are user `admin` and password `Harbor12345`. Additional optional variables (metrics URL, LDAP-related settings, etc.) are documented in [`testutils.py`](testutils.py).

## 3. Run one test module

```bash
export SWAGGER_CLIENT_PATH=./harborclient
export HARBOR_HOST=<your-harbor-host>
# Optional when the test uses the registry:
export DOCKER_USER=<registry-user>
export DOCKER_PWD=<registry-password>

python3 tests/apitests/python/test_verify_metrics_enabled.py
```

Equivalent using `unittest`:

```bash
python3 -m unittest tests.apitests.python.test_verify_metrics_enabled
```

Many modules depend on optional integrations (LDAP, external registries, scanners). Use `unittest` discovery only when your Harbor environment matches what those tests expect:

```bash
python3 -m unittest discover -s tests/apitests/python -p 'test_*.py'
```
