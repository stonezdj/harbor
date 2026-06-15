Harbor v2.15.2 Test Cases
=======

# Purpose:

To verify the major changes included in Harbor v2.15.2 according to the following PRs:

* https://github.com/goharbor/harbor/pull/23379 - Upgrade Harbor UI to Angular 21, Clarity v18, and Node.js v22.
* https://github.com/goharbor/harbor/pull/23378 - Replace direct `gopkg.in/yaml.v2` usage with `github.com/goccy/go-yaml`.
* https://github.com/goharbor/harbor/pull/23356 - Harden crypto usage, password hashing, JWT signing, PostgreSQL auth, and remove unused SMTP package.
* https://github.com/goharbor/harbor/pull/23328 - Upgrade Go OSS packages, AWS SDK v2, pgx v5, Kubernetes libraries, and related dependencies.

# References:

Harbor v2.15.2 release validation, PR descriptions, and Harbor user guide.

# Environment:

* A clean Harbor v2.15.2 instance is running and available.
* An upgraded Harbor instance is available for upgrade validation from a previous v2.15.x build.
* A Linux host with Docker CLI, ORAS CLI, Helm CLI, curl, jq, openssl, and psql installed.
* Admin and non-admin local DB users are available.
* Trivy is enabled.
* Optional external services are available for related cases: AWS ECR test registry, OIDC provider, LDAP server, and a second Harbor instance for replication.
* Browser developer tools can be used to check console errors and network errors.

# Test Cases:

## Test 15.2-01 - Fresh install and service health

# Purpose:

To verify Harbor v2.15.2 can be installed from a clean environment after the dependency, PostgreSQL driver, and portal build changes.

# Test Steps:

1. Install Harbor v2.15.2 with the standard installer.
2. Start Harbor.
3. Run `docker compose ps` from the Harbor installation directory.
4. Access `https://<harbor_host>/api/v2.0/health`.
5. Log in to the UI as admin.
6. Verify core pages can be loaded: Projects, Logs, Registries, Configuration, and Replications.
7. Check `core`, `jobservice`, `portal`, `registryctl`, `registry`, `database`, and `redis` logs for startup errors.

# Expected Outcome:

* All containers are healthy.
* Health API returns healthy status.
* Admin can log in to the UI.
* No panic, migration failure, YAML parse failure, PostgreSQL driver error, or portal static asset error is found in logs.

## Test 15.2-02 - Upgrade from previous v2.15.x build

# Purpose:

To verify database schema migration, pgx v5 migration driver, existing data, and UI compatibility after upgrade.

# Test Steps:

1. Deploy a previous v2.15.x Harbor build.
2. Create one project, one local user, one robot account, one registry endpoint, one replication rule, one retention rule, and one webhook policy.
3. Push at least one image and one ORAS artifact.
4. Upgrade the deployment to Harbor v2.15.2.
5. Start Harbor and wait until all services are healthy.
6. Log in as admin and verify all objects created before upgrade still exist.
7. Log in as the pre-upgrade local user.
8. Pull the pre-upgrade image using Docker CLI.
9. Pull the pre-upgrade ORAS artifact using ORAS CLI.
10. Check migration and core logs.

# Expected Outcome:

* Upgrade completes successfully.
* Existing projects, users, robot accounts, artifacts, policies, and registry endpoints are preserved.
* Existing local users can still log in.
* Existing images and ORAS artifacts can still be pulled.
* No database migration or pgx driver error is found.

## Test 15.2-03 - Portal build, Swagger UI, and static assets

# Purpose:

To verify the Angular 21, Clarity v18, Node.js v22, and Swagger UI bundle changes do not break portal rendering or API documentation.

# Test Steps:

1. Open the Harbor UI in a browser.
2. Open browser developer tools and keep the Console and Network tabs visible.
3. Log in as admin.
4. Navigate to `Administration -> API Explorer`.
5. Verify the Swagger UI page loads.
6. Expand several API groups, such as Project, Repository, Artifact, User, and Robot.
7. Execute a simple GET API from Swagger UI, such as list projects.
8. Refresh the page.
9. Open the direct Swagger asset URLs:
   * `https://<harbor_host>/swagger-ui-index.html`
   * `https://<harbor_host>/swagger-ui.bundle.js`
10. Check browser console and network results.

# Expected Outcome:

* Portal loads successfully.
* Swagger UI renders without a blank page.
* No `TypeError: i is not a function`, React `createRoot`, missing asset, or uncaught Angular error appears.
* API requests from Swagger UI work with the logged-in session.
* Direct Swagger assets return HTTP 200.

## Test 15.2-04 - UI regression smoke test after Angular and Clarity upgrade

# Purpose:

To verify major Harbor UI workflows still work after template migration and Clarity component changes.

# Test Steps:

1. Log in as admin.
2. Create a project.
3. Open project detail page and switch between Repositories, Members, Labels, Robot Accounts, Logs, Configuration, Webhooks, Scanner, Policy, and P2P Preheat tabs.
4. Use search, filter, refresh, pagination, row selection, and action dropdowns on pages that provide them.
5. Open modals for create, edit, delete, confirmation, and details operations.
6. Toggle between light and dark theme from account settings.
7. Resize the browser to desktop and narrow mobile-like widths.
8. Check all pages for clipped text, broken dropdown overlay, invisible icons, invalid input styling, and console errors.

# Expected Outcome:

* All project tabs and administration pages render correctly.
* Modals and confirmation dialogs open and close correctly.
* Dropdowns, datagrids, filters, and pagination work.
* Light and dark themes apply correctly.
* No layout overlap, broken icon, missing style, or Angular runtime error appears.

## Test 15.2-05 - User registration, sign-in, password hashing, and password update

# Purpose:

To verify new and updated local DB passwords use the new PBKDF2-SHA256 version while legacy users can still authenticate.

# Test Steps:

1. Enable self-registration in DB auth mode.
2. Register a new user from the UI.
3. Log out and log in as the new user by username.
4. Log out and log in as the same user by email.
5. Run `docker login <harbor_host>` with the new user's credentials.
6. As the user, change the password from account settings.
7. Log out and log in again with the new password.
8. Run `docker login <harbor_host>` with the new password.
9. As admin, create another local user from the UI.
10. Log in as the admin-created user and change its password.
11. If database access is allowed, verify the affected users have `password_version` set to `pbkdf2_sha256`.
12. For an upgraded environment, log in with a pre-upgrade user whose password version is `sha1` or `sha256`.
13. Change the pre-upgrade user's password and verify login still works.

# Expected Outcome:

* New users can log in through UI and Docker CLI.
* Updated passwords work for UI and Docker CLI login.
* New or changed local users use `pbkdf2_sha256`.
* Existing legacy users can still authenticate before password change.
* Legacy users move to the new password version after password change.
* Invalid old password, weak password, and mismatched confirmation are rejected with clear UI errors.

## Test 15.2-06 - Robot account authentication and token issuance

# Purpose:

To verify token generation remains compatible after JWT signing restrictions and password hashing changes.

# Test Steps:

1. Log in as admin.
2. Create a project.
3. Create a project robot account with push and pull permissions.
4. Copy the robot secret.
5. Run `docker login <harbor_host>` with the robot account and secret.
6. Push an image to the project with Docker CLI.
7. Pull the image with Docker CLI.
8. Regenerate the robot secret from the UI.
9. Verify the old secret no longer works.
10. Verify the new secret works for Docker login and pull.
11. Check core logs for token signing errors.

# Expected Outcome:

* Robot account can authenticate with the generated secret.
* Push and pull operations succeed according to robot permissions.
* Regenerated secret invalidates the old secret.
* No JWT signing, token parse, or unexpected authentication error appears.

## Test 15.2-07 - Registry token signing method hardening

# Purpose:

To verify the default RSA token signing path works and non-RSA signing methods are rejected.

# Test Steps:

1. In a normal v2.15.2 deployment, run Docker login, push, and pull against Harbor.
2. Confirm registry access works with the default token signing configuration.
3. In a controlled negative-test environment, configure Harbor to use a non-RSA JWT signing method if the deployment tooling exposes that option.
4. Restart Harbor core.
5. Attempt to start core and request a registry token.
6. Restore the default RSA signing configuration.
7. Restart Harbor core and repeat Docker login, push, and pull.

# Expected Outcome:

* Default RS256 token signing works.
* Non-RSA signing methods are rejected during option creation or token service startup.
* Harbor does not silently accept unsupported signing methods.
* Restoring RSA signing returns registry authentication to normal.

## Test 15.2-08 - PostgreSQL SCRAM authentication on fresh install

# Purpose:

To verify PostgreSQL initialization uses `scram-sha-256` password authentication and Harbor services can connect.

# Test Steps:

1. Install Harbor v2.15.2 with a configured database password.
2. Start Harbor.
3. Check the PostgreSQL `pg_hba.conf` inside the database container.
4. Confirm password authentication entries use `scram-sha-256`.
5. Log in to Harbor as admin.
6. Create a project and push an image.
7. Restart the database container.
8. Wait until all Harbor services reconnect.
9. Repeat UI login and image pull.

# Expected Outcome:

* PostgreSQL is initialized with `scram-sha-256`.
* Harbor services connect to PostgreSQL successfully.
* Harbor recovers after database restart.
* No authentication method or database connection error appears.

## Test 15.2-09 - YAML parsing for jobservice and registryctl configuration

# Purpose:

To verify replacing direct YAML library usage does not break Harbor config parsing.

# Test Steps:

1. Start Harbor v2.15.2 with the default generated configuration.
2. Check `jobservice` and `registryctl` logs for config parse errors.
3. Trigger a jobservice task by manually scanning an artifact.
4. Trigger a jobservice task by running garbage collection in dry-run mode.
5. Push and pull an image to exercise registryctl and registry configuration.
6. Restart `jobservice` and `registryctl`.
7. Repeat scan and image pull.

# Expected Outcome:

* `jobservice` and `registryctl` start without YAML parse errors.
* Scan and garbage collection jobs can be scheduled and completed.
* Image push and pull work before and after restart.
* No behavior regression is observed from the YAML library replacement.

## Test 15.2-10 - Helm chart YAML parsing

# Purpose:

To verify chart metadata parsing still works after the YAML library replacement in chart code.

# Test Steps:

1. Prepare a Helm chart with valid `Chart.yaml`, `values.yaml`, and templates.
2. Package the chart with `helm package`.
3. Push or upload the chart to Harbor using the supported Helm chart workflow for the deployment.
4. View the chart in Harbor UI.
5. Download or pull the chart.
6. Modify the chart with common YAML features, such as quoted strings, multiline values, nested maps, lists, and comments.
7. Package and upload the modified chart.
8. Verify the chart metadata, version, app version, and dependencies are displayed correctly.
9. Try uploading a chart with invalid YAML in `Chart.yaml`.

# Expected Outcome:

* Valid charts are accepted.
* Chart metadata is parsed and displayed correctly.
* Downloaded chart matches the uploaded chart.
* Invalid chart YAML is rejected with a clear error.

## Test 15.2-11 - AWS ECR registry endpoint and replication

# Purpose:

To verify AWS SDK v2 changes in the AWS ECR adapter, credential handling, custom CA handling, endpoint override, and error handling.

# Test Steps:

1. Prepare an AWS ECR repository and credentials with pull permission.
2. Log in to Harbor as admin.
3. Go to `Administration -> Registries`.
4. Add an AWS ECR endpoint with valid region, access key, secret key, and registry URL.
5. Click `Test Connection`.
6. Save the endpoint.
7. Create a pull-based replication rule from the ECR endpoint to a Harbor project.
8. Trigger replication manually.
9. Verify the image appears in Harbor after replication.
10. Pull the replicated image from Harbor.
11. Edit the endpoint with an invalid secret and click `Test Connection`.
12. If available, repeat with a custom endpoint override or self-signed CA scenario.
13. Check jobservice and core logs.

# Expected Outcome:

* Valid AWS ECR endpoint test succeeds.
* Replication from ECR succeeds.
* Replicated image can be pulled from Harbor.
* Invalid credentials fail with a meaningful AWS error.
* No token cache panic, nil pointer, or leaked credential value appears in logs.

## Test 15.2-12 - Generic registry endpoints and replication regression

# Purpose:

To verify registry adapter and dependency upgrades did not break non-AWS registry endpoints.

# Test Steps:

1. Prepare a second Harbor instance or another supported registry.
2. Log in to Harbor as admin.
3. Add the remote registry endpoint with valid credentials.
4. Test connection and save.
5. Create a pull replication rule from the remote registry.
6. Trigger replication manually.
7. Create a push replication rule to the remote registry.
8. Trigger push replication manually.
9. Verify replication executions, tasks, logs, and artifacts on both sides.
10. Repeat endpoint test with invalid URL and invalid credentials.

# Expected Outcome:

* Valid endpoint connection succeeds.
* Pull and push replication jobs complete successfully.
* Invalid endpoint cases fail cleanly.
* Replication logs are readable and contain no dependency-related error.

## Test 15.2-13 - ORAS artifact push and pull with SHA256 verification

# Purpose:

To verify ORAS artifact workflows and SHA256 checksum validation after test and dependency updates.

# Test Steps:

1. Create a private project.
2. Log in to Harbor with ORAS CLI.
3. Create two local files, such as `artifact.txt` and `readme.md`.
4. Record SHA256 checksums for both files using `sha256sum`.
5. Push the files as an ORAS artifact to Harbor.
6. Open the artifact in Harbor UI and verify the artifact and tag are visible.
7. Pull the ORAS artifact into a clean directory.
8. Record SHA256 checksums for the pulled files.
9. Compare pushed and pulled SHA256 values.
10. Delete the artifact from Harbor UI.

# Expected Outcome:

* ORAS push succeeds.
* Artifact is visible in Harbor UI.
* ORAS pull succeeds.
* Pushed and pulled SHA256 checksums are identical.
* Artifact deletion succeeds.

## Test 15.2-14 - SBOM and checksum display

# Purpose:

To verify SBOM parsing and checksum display after SHA256-related updates.

# Test Steps:

1. Push an image to a project with Trivy enabled.
2. Generate SBOM for the artifact from Harbor UI or API.
3. Wait until the SBOM generation job succeeds.
4. Open the artifact SBOM tab or download the SBOM.
5. Verify package checksum fields use valid SHA256 values where present.
6. Download the SBOM file.
7. Validate the SBOM JSON structure with a JSON parser.
8. Check jobservice logs for SBOM generation errors.

# Expected Outcome:

* SBOM generation succeeds.
* SBOM data is displayed and downloadable.
* SHA256 checksum values are complete and valid.
* No UI render error or SBOM job error appears.

## Test 15.2-15 - P2P preheat provider and policy UI

# Purpose:

To verify the Angular upgrade and dependency changes do not break P2P preheat provider and policy workflows.

# Test Steps:

1. Log in as admin.
2. Create a project.
3. Go to the project `P2P Preheat` tab.
4. Add a supported preheat provider with valid endpoint and credentials.
5. Create a preheat policy with filters and a trigger.
6. Push an image that matches the policy.
7. Trigger the policy manually if supported.
8. Verify preheat execution status and logs.
9. Edit the policy and save.
10. Delete the policy.
11. Add a provider with invalid endpoint or credentials and test validation.

# Expected Outcome:

* Provider and policy forms render correctly.
* Valid provider and policy can be created, edited, triggered, and deleted.
* Preheat execution status is shown correctly.
* Invalid provider validation is shown clearly.
* No Angular `ExpressionChangedAfterItHasBeenCheckedError` or Clarity modal/dropdown issue appears.

## Test 15.2-16 - Audit log purge job status

# Purpose:

To verify audit log purge job behavior and the accepted `Stopped` or `Success` dry-run status.

# Test Steps:

1. Log in as admin.
2. Generate several audit log records by logging in, creating a project, pushing an image, and deleting a tag.
3. Go to the audit log purge configuration page.
4. Create a manual dry-run purge job.
5. Immediately stop the dry-run job.
6. Refresh the purge history.
7. Create a manual non-dry-run purge job with a valid retention hour.
8. Wait for the job to complete.
9. View job details and logs.

# Expected Outcome:

* Dry-run purge job status becomes either `Stopped` or `Success`.
* Non-dry-run purge job completes successfully.
* Purge history is displayed correctly.
* Job logs are available.

## Test 15.2-17 - OIDC certificate verification and TLS behavior

# Purpose:

To verify OIDC login still works and the insecure certificate fallback requires TLS 1.2 or newer.

# Test Steps:

1. Configure Harbor to use an OIDC provider with a valid certificate.
2. Log in through OIDC.
3. Configure Harbor to use an OIDC provider with a self-signed certificate.
4. Enable the setting that disables OIDC certificate verification.
5. Log in through OIDC again.
6. If a test OIDC provider with TLS 1.0 or TLS 1.1 is available, configure it and attempt login.
7. Restore the valid OIDC configuration.

# Expected Outcome:

* OIDC login with valid certificate succeeds.
* OIDC login with self-signed certificate succeeds only when certificate verification is disabled.
* TLS 1.0 or TLS 1.1 OIDC endpoint is rejected.
* No unexpected panic or token validation error appears.

## Test 15.2-18 - LDAP and auth dependency regression

# Purpose:

To verify upgraded auth-related dependencies do not break LDAP login or LDAP group membership.

# Test Steps:

1. Configure Harbor to use LDAP auth.
2. Test LDAP server connection from the UI.
3. Log in as an LDAP user.
4. Add an LDAP group to a project.
5. Log in as a user from that LDAP group.
6. Verify project permissions from the group are applied.
7. Search LDAP users and groups from project member management.
8. Configure invalid LDAP bind credentials and test connection.

# Expected Outcome:

* Valid LDAP connection succeeds.
* LDAP users can log in.
* LDAP group project permissions work.
* LDAP user and group search works.
* Invalid LDAP configuration fails with a clear error.

## Test 15.2-19 - Scanner, vulnerability, and Kubernetes dependency regression

# Purpose:

To verify scanner, scan report, and Kubernetes-related dependency upgrades do not break vulnerability workflows.

# Test Steps:

1. Log in as admin.
2. Confirm the default scanner is configured.
3. Create a project and push a test image.
4. Manually scan the image.
5. Enable scan on push for the project.
6. Push another tag of the image.
7. Verify automatic scan starts.
8. Open vulnerability details, CVE allowlist, and scan report download.
9. Configure a severity deployment security policy and attempt to pull an image above the threshold.

# Expected Outcome:

* Manual scan succeeds.
* Scan on push succeeds.
* Vulnerability details and reports render correctly.
* Security policy blocks or allows pulls according to configuration.
* No scanner adapter or report parsing error appears.

## Test 15.2-20 - Security and dependency negative checks

# Purpose:

To verify the hardening changes do not introduce unsafe fallbacks or behavior regressions.

# Test Steps:

1. Try to log in with an incorrect password for a local user.
2. Try to use an old robot secret after regenerating the secret.
3. Try to access a private project artifact without authentication.
4. Try Docker pull with a token generated for another project.
5. Try an invalid registry endpoint with malformed URL and invalid credentials.
6. Try uploading malformed chart YAML.
7. Try loading Swagger UI in a logged-out browser session and execute an authenticated API.
8. Review logs for stack traces, leaked secrets, or unexpected successful authentication.

# Expected Outcome:

* Incorrect credentials and invalid tokens are rejected.
* Private artifacts remain protected.
* Invalid registry endpoint and invalid chart YAML fail cleanly.
* Swagger UI does not allow authenticated API execution without login.
* Logs do not leak passwords, robot secrets, AWS secrets, or registry tokens.

# Possible Problems:

* Some optional cases require external services, such as AWS ECR, LDAP, or OIDC.
* Token signing negative test depends on whether the deployment exposes signing method configuration.
* Helm chart workflow depends on the chart feature set enabled in the tested Harbor deployment.
