**Description:** Verify that proxying the OCI 1.1 referrers API is disabled by default. When enabled, Harbor returns upstream referrers merged with locally generated referrers, such as SBOMs.

**Steps**

Complete the categories in order. Use an authenticated HTTP client for API requests and follow any next-page links when comparing complete referrer lists.

### 1. Environment and test data

1. Install the Harbor v2.16.0-rc1. Log in as the administrator and keep automatic SBOM generation disabled.
2. Prepare an upstream registry supporting the OCI 1.1 referrers API (Harbor v2.15.0). Push an image and attach an OCI signature referring to its digest. Record the image digest as `<subject-digest>` and the signature's digest and artifact type. Verify that the upstream referrers API returns the signature.
3. Go to Administration → Registries, add the upstream endpoint, and verify that the connection succeeds. In the following steps, `<proxy-repository>` means `referrer-proxy/<upstream-repository>`, including any upstream namespace.

### 2. Default behavior

1. Go to Projects → New Project, enable Proxy Cache, and select the upstream endpoint. Verify that **Enable proxy for referrer API** is unchecked by default. Create a project named `referrer-proxy` without changing this option. Do not pull the image or signature yet.
2. Open the project's Configuration tab and verify that the option remains unchecked. Request `GET /v2/<proxy-repository>/referrers/<subject-digest>`. Verify HTTP 200 with an empty `manifests` array because no referrers exist locally. The upstream signature must not appear.

### 3. Enable upstream referrer proxy cache

1. In the Configuration tab, select **Enable proxy for referrer API**, save, and refresh the page. Verify that the option remains enabled.
2. Repeat `GET /v2/<proxy-repository>/referrers/<subject-digest>`. Verify HTTP 200 with `Content-Type: application/vnd.oci.image.index.v1+json`. Verify that the returned signature's digest and artifact type match the upstream result, even though the image and signature have not been pulled into Harbor.

### 4. Merge locally generated referrers

1. Pull the subject image through `referrer-proxy`. In Harbor, select the image and run **Generate SBOM**. Wait for completion and record the local SBOM's digest and artifact type. Do not pull the upstream signature.
2. Request the Harbor referrers API again. Verify that the result contains both the upstream signature and the locally generated SBOM, each exactly once. Query the upstream directly and verify that it still contains only the signature. Verify that the local SBOM appears only on the last.
3. Repeat the Harbor request with `?artifactType=<encoded-local-sbom-artifact-type>`, then with the signature artifact type. Verify that each result contains only the matching referrer.
4. Disable **Enable proxy for referrer API**, save, and repeat the unfiltered request. Verify that Harbor returns only the locally generated SBOM; the upstream signature must not appear.
5. Re-enable the option and repeat the request. Verify that Harbor again returns both the upstream signature and the local SBOM without duplicates.
