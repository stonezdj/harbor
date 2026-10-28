**Description:** Verify Harbor installation on Linux arm64 using offline and online installers, including image push/pull, Trivy scanning, and exporter metrics.

**Reference:** [PR #22311: Full Multi-Architecture Enablement for Harbor (amd64 + arm64)](https://github.com/goharbor/harbor/pull/22311).

**Steps**

Use a clean arm64 host or restore a clean snapshot for each installation method. Complete category 4 after each installation.

### 1. Environment setup

1. Prepare a native Linux arm64 host with Docker and Docker Compose installed. Run `uname -m` and verify that it returns `aarch64`. Use a native Docker daemon without CPU emulation or a `DOCKER_DEFAULT_PLATFORM` override.
2. Download `harbor-offline-installer-<version>-arm64.tgz` and `harbor-online-installer-<version>-arm64.tgz` for a Harbor build containing PR #22311. Record the build version. Prepare HTTPS certificates and a small runnable Linux arm64 image for push/pull verification.

### 2. Offline installation

1. Extract the offline installer and copy `harbor.yml.tmpl` to `harbor.yml`. Configure the hostname, HTTPS certificates, administrator password, and data directory. Enable metrics with `metric.enabled: true`, port `9090`, and path `/metrics`.
2. Confirm that the host has no preloaded Harbor images. Block outbound access to external container registries while retaining local access to Harbor, then run `sudo ./install.sh --with-trivy`. Verify that installation loads the required images from the package and completes without downloading component images.
3. Restore network access so Trivy can download its vulnerability databases, then complete category 4.

### 3. Online installation

1. On a clean arm64 host with no Harbor images, extract the online installer and configure `harbor.yml` as in category 2, step 1.
2. With registry access available, run `sudo ./install.sh --with-trivy`. Verify that the required images download and installation completes without changing image tags or specifying `--platform`. Complete category 4.

### 4. Installation and feature verification

1. Run `docker compose ps` and request `GET /api/v2.0/health`. Wait for startup to finish and verify that all configured services are running and healthy. Check the logs for startup failures or `exec format error` messages; none should occur.
2. For each Harbor component image, including Trivy and the exporter, run `docker image inspect --format '{{.Os}}/{{.Architecture}}' <image>`. Verify that every image reports `linux/arm64`.
3. Open Harbor and log in as `admin`. Create a private project named `arm64-test`, run `docker login`, and push the prepared arm64 image to `arm64-test/smoke:latest`. Verify that the repository, tag, and digest appear correctly in the portal.
4. Pull the image from an arm64 client without a cached copy. Verify that its digest matches the pushed artifact and that the image runs successfully.
5. Select the artifact in the portal and click **Scan**. Wait for completion and verify that the Trivy scan report is available without architecture-related errors.
6. Request `http://<harbor-host>:9090/metrics`. Verify HTTP 200 and valid Prometheus metrics from the Harbor exporter.
7. Restart Harbor while retaining its data. Verify healthy services, successful login, the existing project and artifact, a successful image pull, and working metrics. Remove the test resources afterward.
