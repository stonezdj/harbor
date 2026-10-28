**Description:** Verify that proxying the OCI 1.1 referrers API is disabled by default. When enabled, Harbor returns upstream referrers merged with locally generated referrers, such as SBOMs.

**Steps**

Complete the categories in order. Use an authenticated HTTP client for API requests and follow any next-page links when comparing complete referrer lists. The non-GET API examples below correspond to the actions in each step; send JSON bodies with `Content-Type: application/json` as the administrator. Replace placeholders and illustrative numeric IDs with values from the test environment. Project metadata flags are strings (`"true"` or `"false"`).

### 1. Environment and test data

1. Install the Harbor v2.16.0-rc1. Log in as the administrator and keep automatic SBOM generation disabled.

   Automatic SBOM generation is configured per project. To disable it for an existing test project, send `PUT /api/v2.0/projects/<project-name>` with the following body. For the new `referrer-proxy` project, set it during creation in category 2, step 1.

   ```json
   {
     "metadata": {
       "auto_sbom_generation": "false"
     }
   }
   ```

2. Prepare an upstream registry supporting the OCI 1.1 referrers API (Harbor v2.15.0). Push an image and attach an OCI signature referring to its digest. Record the image digest as `<subject-digest>` and the signature's digest and artifact type. Verify that the upstream referrers API returns the signature.
3. Go to Administration → Registries, add the upstream endpoint, and verify that the connection succeeds. In the following steps, `<proxy-repository>` means `referrer-proxy/<upstream-repository>`, including any upstream namespace.

   To add the endpoint through the API, send `POST /api/v2.0/registries` with the following body. Replace the URL and credentials with those of the upstream Harbor registry. Record its numeric ID from the successful response's `Location` header.

   ```json
   {
     "name": "referrer-upstream",
     "type": "harbor",
     "url": "https://<upstream-harbor-host>",
     "insecure": false,
     "credential": {
       "type": "basic",
       "access_key": "<upstream-username>",
       "access_secret": "<upstream-password>"
     }
   }
   ```

   To test the saved endpoint's connection, send `POST /api/v2.0/registries/ping` with the following body, replacing `1` with the recorded registry ID. Verify HTTP 200.

   ```json
   {
     "id": 1
   }
   ```

### 2. Default behavior

1. Go to Projects → New Project, enable Proxy Cache, and select the upstream endpoint. Verify that **Enable proxy for referrer API** is unchecked by default. Create a project named `referrer-proxy` without changing this option. Do not pull the image or signature yet.

   For API creation, send `POST /api/v2.0/projects` with the following body, replacing `registry_id` with the upstream endpoint's numeric ID. Omit `proxy_referrer_api` to exercise its disabled default and explicitly disable automatic SBOM generation.

   ```json
   {
     "project_name": "referrer-proxy",
     "registry_id": 1,
     "metadata": {
       "public": "false",
       "auto_sbom_generation": "false"
     }
   }
   ```

2. Open the project's Configuration tab and verify that the option remains unchecked. Request `GET /v2/<proxy-repository>/referrers/<subject-digest>`. Verify HTTP 200 with an empty `manifests` array because no referrers exist locally. The upstream signature must not appear.

### 3. Enable upstream referrer proxy cache

1. In the Configuration tab, select **Enable proxy for referrer API**, save, and refresh the page. Verify that the option remains enabled.

   To enable it through the API, send `PUT /api/v2.0/projects/referrer-proxy` with:

   ```json
   {
     "metadata": {
       "proxy_referrer_api": "true"
     }
   }
   ```

2. Repeat `GET /v2/<proxy-repository>/referrers/<subject-digest>`. Verify HTTP 200 with `Content-Type: application/vnd.oci.image.index.v1+json`. Verify that the returned signature's digest and artifact type match the upstream result, even though the image and signature have not been pulled into Harbor.

### 4. Merge locally generated referrers

1. Pull the subject image through `referrer-proxy`. In Harbor, select the image and run **Generate SBOM**. Wait for completion and record the local SBOM's digest and artifact type. Do not pull the upstream signature.

   To generate the SBOM through the API after pulling the subject image, send `POST /api/v2.0/projects/referrer-proxy/repositories/<encoded-upstream-repository>/artifacts/<subject-digest>/scan` with the following body. Verify HTTP 202 and wait for generation to finish. The repository path excludes the `referrer-proxy/` project prefix; double-encode slashes in the upstream repository name, for example `library/alpine` becomes `library%252Falpine`.

   ```json
   {
     "scan_type": "sbom"
   }
   ```

2. Request the Harbor referrers API again. Verify that the result contains both the upstream signature and the locally generated SBOM, each exactly once. Query the upstream directly and verify that it still contains only the signature. Verify that the local SBOM appears only on the last.
3. Repeat the Harbor request with `?artifactType=<encoded-local-sbom-artifact-type>`, then with the signature artifact type. Verify that each result contains only the matching referrer.
4. Disable **Enable proxy for referrer API**, save, and repeat the unfiltered request. Verify that Harbor returns only the locally generated SBOM; the upstream signature must not appear.

   Send `PUT /api/v2.0/projects/referrer-proxy` with:

   ```json
   {
     "metadata": {
       "proxy_referrer_api": "false"
     }
   }
   ```

5. Re-enable the option and repeat the request. Verify that Harbor again returns both the upstream signature and the local SBOM without duplicates.

   Send `PUT /api/v2.0/projects/referrer-proxy` with:

   ```json
   {
     "metadata": {
       "proxy_referrer_api": "true"
     }
   }
   ```
