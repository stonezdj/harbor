**Description:** Verify that proxy cache projects can restrict upstream repositories using Doublestar or regular expression filters. Verify the repository filter pattern and matching kind through the UI, project API, and metadata API, including defaults, validation, persistence, clearing, permissions, and enforcement on registry requests.

**Reference:** [PR #23527](https://github.com/goharbor/harbor/pull/23527) and [PR #23536](https://github.com/goharbor/harbor/pull/23536).

**Steps**

Complete the categories in order. Authenticate Harbor management API requests as the system administrator unless another user is specified. Send JSON bodies with `Content-Type: application/json`. Replace placeholders and illustrative registry ID `1` with values from the test environment. Metadata values are strings. For `/v2/` requests, use a registry client that completes Harbor's bearer-token authentication, or obtain a token with pull scope for the requested repository. Follow redirects and check the final HTTP response; an authentication failure is not a filter result. Include the appropriate OCI/Docker manifest `Accept` headers for manifest requests.

For UI checks, use the browser's Network panel to inspect the filter fields in creation and update requests. Use separate browser sessions for the system administrator and ordinary users.

### 1. Environment and test data

1. Install Harbor v2.16.0-rc1 containing PRs #23527 and #23536 and log in as the system administrator. Use English as the UI language. Enable database audit logging and the `update_project` event. Record the original audit settings for restoration.
2. Prepare a reachable upstream Harbor registry supporting the OCI 1.1 referrers API. Push small single-platform images tagged `v1` into `library/alpine`, `library/nginx`, `library/team/app`, and `other/busybox`. Verify direct pulls of all four images. Record each image manifest digest and a blob digest from its manifest. These are controlled test repositories; their names do not require using the corresponding public images.
3. Attach an OCI referrer to the images in `library/alpine` and `other/busybox`. Verify that upstream referrer requests return the attached artifacts. Keep upstream access logs available to check whether blocked requests reach it.
4. Go to Administration → Registries, add the upstream endpoint, test its connection, and record its numeric ID. Run `docker login <harbor-host>`. In the steps below, a pull means `docker pull <harbor-host>/<proxy-project>/<upstream-repository>:v1`; the filter matches only `<upstream-repository>`, including its upstream namespace.

### 2. Default behavior and project creation

1. Send `POST /api/v2.0/projects` with the following body, omitting both filter fields. Verify HTTP 201 and record the project ID.

   ```json
   {
     "project_name": "filter-default",
     "registry_id": 1,
     "metadata": {"public": "false"}
   }
   ```

2. Read `GET /api/v2.0/projects/filter-default` and verify that no nonempty repository filter is configured. Pull all four test images through `filter-default`; all must succeed.
3. Create a separate project using the following body. Verify HTTP 201. Read `GET /api/v2.0/projects/filter-api` and `GET /api/v2.0/projects/filter-api/metadatas/`; verify that both return pattern `library/**` and kind `doublestar`.

   ```json
   {
     "project_name": "filter-api",
     "registry_id": 1,
     "metadata": {
       "public": "false",
       "proxy_cache_filter_pattern": "library/**",
       "proxy_cache_filter_kind": "doublestar",
       "proxy_referrer_api": "true"
     }
   }
   ```

4. Pull `library/alpine:v1`, `library/nginx:v1`, and `library/team/app:v1` through `filter-api`; all must succeed. Pull `other/busybox:v1` through the same project; it must fail with a not-found response. Confirm that the denied image is not added to this project's repository list and that the upstream receives no request for this denied pull.

### 3. Registry API enforcement

1. With `filter-api` still using `library/**`, send `GET` and `HEAD` to `/v2/filter-api/library/alpine/manifests/v1`, then to the same endpoint using the recorded manifest digest instead of `v1`. Verify HTTP 200 and the expected manifest digest.
2. Repeat both methods for `/v2/filter-api/other/busybox/manifests/v1` and its known manifest digest. Verify HTTP 404 for every request, including requests by digest.
3. Send `GET /v2/filter-api/library/alpine/blobs/<alpine-blob-digest>` and verify that the blob is returned. Send `GET /v2/filter-api/other/busybox/blobs/<busybox-blob-digest>` and verify HTTP 404, even though the blob was cached through `filter-default`.
4. Send `GET /v2/filter-api/library/alpine/tags/list` and verify HTTP 200 with tag `v1`. Send `GET /v2/filter-api/other/busybox/tags/list` and verify HTTP 404.
5. Send `GET /v2/filter-api/library/alpine/referrers/<alpine-manifest-digest>` and verify HTTP 200 containing its upstream referrer. Send the equivalent request for `other/busybox` and verify HTTP 404. Keep `proxy_referrer_api` enabled for this check so that the request exercises upstream referrer proxying.
6. Inspect upstream access logs for steps 2–5. Verify that none of the denied manifest, blob, tag-list, or referrer requests reached the upstream registry. The matching requests must still work.

### 4. Pattern matching, updates, and cached content

Use `PUT /api/v2.0/projects/filter-api` to update both filter fields together. For each update, expect HTTP 200 and read the project back to verify the saved values. For example:

```json
{
  "metadata": {
    "proxy_cache_filter_pattern": "library/*",
    "proxy_cache_filter_kind": "doublestar"
  }
}
```

1. Set kind `doublestar` and pattern `library/*`. Verify that `library/alpine:v1` and `library/nginx:v1` can be pulled, while `library/team/app:v1` and `other/busybox:v1` are denied. A single `*` must not cross a path separator.
2. Change the pattern to `library/**`. Verify that `library/team/app:v1` can now be pulled and that `other/busybox:v1` remains denied. Change it to `library/{alpine,nginx}`; verify that only the two named repositories are allowed among the four fixtures.
3. Set kind `regex` and pattern `library/(alpine|nginx)`. Verify that the two named repositories are allowed and the other two are denied. Change the pattern to `alpine`; verify that `library/alpine:v1` is denied because regex matching requires the full upstream repository path.
4. Set kind `doublestar` and pattern `  library/**  `, including leading and trailing spaces. Verify that matching behaves like `library/**`. Set the pattern to three spaces, then to an empty string; verify that each allows all four repositories. Pull `other/busybox:v1` while the empty pattern is active and wait until it is cached in `filter-api`.
5. Restore `library/**` with kind `doublestar`. Repeat the denied requests for `other/busybox` from category 3. They must still return HTTP 404 even though that repository is now cached in this project. Confirm that the repository remains visible to the administrator in the management UI; changing the filter must not delete cached artifacts.
6. Send `PUT /api/v2.0/projects/filter-api` with the body below. Verify HTTP 200, confirm that the filter remains `library/**`, and repeat the denied pull. Enabling stale-content fallback must not allow a blocked repository. Pull the same image through `filter-default` and verify success, demonstrating that the filter is scoped to its project.

   ```json
   {
     "metadata": {
       "proxy_cache_filter_pattern": "library/**",
       "proxy_cache_filter_kind": "doublestar",
       "proxy_cache_local_on_not_found": "true"
     }
   }
   ```

### 5. Invalid configuration

1. For each row below, send `POST /api/v2.0/projects` using the creation body from category 2, step 3, a fresh project name, and the indicated filter values. Verify HTTP 400 with a validation error and confirm that no project was created.

   | Pattern | Kind | Invalid input |
   | --- | --- | --- |
   | `library/[abc` | `doublestar` | Unclosed character class |
   | `library/{alpine,nginx` | `doublestar` | Unclosed alternatives |
   | `[` | `regex` | Invalid regular expression |
   | `library/**` | `glob` | Unsupported matching kind |

2. Repeat each invalid pair with `PUT /api/v2.0/projects/filter-api`, using the metadata body format in category 4. Verify HTTP 400. After each rejection, read the project and confirm that the previous valid filter is unchanged; matching pulls must succeed and blocked pulls must remain denied.
3. Submit the valid pair `library/**` and `doublestar` again. Verify HTTP 200 and successful pulls of matching repositories, confirming that failed validation does not prevent later valid updates.

### 6. Individual metadata operations

1. Create `filter-meta` using the unfiltered creation body from category 2, step 1. Send `POST /api/v2.0/projects/filter-meta/metadatas/` with `{"proxy_cache_filter_kind":"doublestar"}`, then a separate POST to the same endpoint with `{"proxy_cache_filter_pattern":"library/**"}`. Verify HTTP 200 for both. Metadata requests contain one key/value pair, without a `metadata` wrapper.
2. Read `GET /api/v2.0/projects/filter-meta/metadatas/proxy_cache_filter_pattern` and the corresponding `proxy_cache_filter_kind` endpoint. Verify the saved values. Send `PUT /api/v2.0/projects/filter-meta/metadatas/proxy_cache_filter_pattern` with `{"proxy_cache_filter_pattern":"library/alpine"}`. Verify HTTP 200, a successful alpine pull, and a denied nginx pull.
3. Update the pattern to `library/**` through the same endpoint. Try changing only the kind with `PUT /api/v2.0/projects/filter-meta/metadatas/proxy_cache_filter_kind` and `{"proxy_cache_filter_kind":"regex"}`. Verify HTTP 400 because the existing pattern is invalid as a regex, and verify that both stored values remain unchanged.
4. Try updating the pattern to `library/[abc` while the stored kind is `doublestar`, then try setting the kind to `glob`. Verify HTTP 400 for each and confirm that the valid stored filter is preserved.
5. Send `DELETE /api/v2.0/projects/filter-meta/metadatas/proxy_cache_filter_kind` while the pattern is nonempty. Verify HTTP 400 and that the kind is still present.
6. Clear the pattern using `PUT /api/v2.0/projects/filter-meta/metadatas/proxy_cache_filter_pattern` with `{"proxy_cache_filter_pattern":""}`. Verify HTTP 200 and successful pulls of all four repositories. Delete the kind, then delete the pattern through their individual metadata endpoints; verify HTTP 200 and that neither key is present in the metadata list.

### 7. Repository filter creation through the UI

1. Open Projects → New Project as the system administrator. Verify that the repository filter controls are hidden while Proxy Cache is off. Enable Proxy Cache and select the upstream endpoint from category 1. Verify that **Repository filter** is empty, **Doublestar** is selected by default, and **Regex** is available. Open the tooltip and verify that it explains the optional filter, `**` matching, and full-path regex matching.
2. Create `filter-ui-default` without entering a filter. Verify that no nonempty filter is sent in the creation request. Open Configuration and refresh; the pattern must remain empty and the kind must display Doublestar. Pull all four fixtures through this project and verify success.
3. Open New Project again, enable Proxy Cache, and select the same endpoint. Enter `library/**` and keep Doublestar selected. Create `filter-ui-glob`. Verify that the POST request contains `metadata.proxy_cache_filter_pattern="library/**"` and `metadata.proxy_cache_filter_kind="doublestar"`. Verify the same values in `GET /api/v2.0/projects/filter-ui-glob` and on Configuration after refreshing.
4. Pull the three `library/` images through `filter-ui-glob`; all must succeed, including `library/team/app:v1`. Pull `other/busybox:v1` and verify that it is denied.
5. Create `filter-ui-regex` with pattern `library/(alpine|nginx)` and kind Regex. Verify that the creation request sends kind `regex` and the exact pattern, and that both persist in the project API and on Configuration after refreshing. Verify that alpine and nginx pulls succeed while the nested app and busybox pulls are denied.

### 8. Repository filter UI validation

1. Open New Project, enter an unused name, enable Proxy Cache, and select the endpoint. Select Regex and enter `[`. Verify an inline invalid-regex error and that **OK** is disabled. Replace it with `library/.*`; verify that the error clears and submission becomes available.
2. Select Doublestar and enter `library/[abc`, then `library/{alpine,nginx`. Verify an inline validation error and disabled OK for each. Correct the pattern to `library/{alpine,nginx}`; verify that the error clears.
3. Enter `library/**` with Doublestar selected, then switch to Regex without changing the text. Verify immediate revalidation and an error for the incompatible regex. Switch back to Doublestar and verify that the error clears. Clear the pattern and verify that an empty optional filter is accepted with either kind.
4. Leave an invalid filter in the dialog and click **Cancel**. Reopen New Project and enable Proxy Cache. Verify that the pattern is empty, Doublestar is selected, and the previous filter error is gone. Enter a new name, select the endpoint, and create `filter-ui-reset` successfully with no filter.
5. Open `filter-ui-glob` → Configuration and repeat the invalid-pattern and kind-switching checks from steps 1–3. Verify inline validation and disabled **Save** for invalid input, with no invalid update request submitted. Correct the filter to `library/alpine` with Doublestar selected, save, and refresh. Verify that validation clears and the corrected filter persists.

### 9. Editing, clearing, and canceling the filter in the UI

1. Open `filter-ui-glob` → Configuration. Change the pattern to `library/*`, save, and refresh. Verify that the update request includes both filter fields and that the project API and UI retain the saved values. Pull alpine and nginx successfully; verify that `library/team/app:v1` is denied despite having been cached earlier.
2. Change the kind to Regex and the pattern to `library/alpine`. Save and refresh. Verify that the request sends kind `regex` and that only alpine is allowed among the four fixtures. Change the pattern to `alpine`, save, and verify that `library/alpine:v1` is denied because a partial repository name is not a full-path match. Restore `library/alpine` and save.
3. Clear the pattern, save, and inspect the request. Verify that `proxy_cache_filter_pattern` is sent as an empty string so the existing filter is removed. Refresh, verify the empty field and the cleared API value, and pull all four fixtures successfully.
4. Restore `library/**` with Doublestar and save. Change the pattern to `library/alpine` and the kind to Regex without saving. Click **Cancel** and confirm discarding the changes. Verify that both filter controls return to their last saved values, no update request is sent, and a page refresh shows the saved filter.
5. Enter an invalid regex, then cancel and confirm. Repeat with an invalid Doublestar pattern. Verify that cancel restores the saved pattern and kind and clears the validation message in each case. Enter the valid pattern `library/nginx` with Doublestar, save, and verify that nginx can be pulled while alpine is denied.

### 10. Normal projects, filter permissions, and Docker Hub paths

1. Create a normal project named `filter-normal` using the creation body from category 2, step 3, but omit `registry_id`. Verify that the project is created without effective proxy filter metadata and that its Configuration page has no repository filter controls. Update it through `PUT /api/v2.0/projects/filter-normal` with the valid metadata body from category 4. Verify success and that the proxy filter fields remain absent or unset.
2. Try adding each filter key through `POST /api/v2.0/projects/filter-normal/metadatas/` and updating each through its individual metadata PUT endpoint. Use the valid one-key bodies from category 6. Verify HTTP 400 because normal projects do not support these metadata operations. Push and pull a test image in `filter-normal` and verify normal behavior.
3. Create two ordinary users and add one as Project Admin and the other as Guest in `filter-api`. Log in as each user and open Configuration. Verify that the saved `library/**` filter appears in a disabled field with a `doublestar` label. Neither user can edit the repository filter through this page.
4. As the Guest, try the valid filter project PUT and metadata PUT requests from categories 4 and 6, targeting `filter-api` and changing its pattern to `other/**`. Verify HTTP 403 and unchanged configuration. Verify that the Guest can pull `library/alpine:v1` but cannot pull `other/busybox:v1`.
5. As the system administrator, open `filter-api` → Configuration, change the pattern to `library/alpine` and kind to Regex, and save. Refresh Configuration in both ordinary-user sessions and verify that the read-only pattern and `regex` label update.
6. Clear the filter as the system administrator and save. Refresh both ordinary-user sessions and verify that the previous filter is no longer displayed. Log back in as the system administrator and verify that the filter controls remain editable. Restore `library/**` with Doublestar and save.
7. Add a Docker Hub endpoint and create `filter-hub` with that endpoint's ID, pattern `library/alpine`, and kind `doublestar`. First verify that selected alpine and nginx tags exist directly on Docker Hub. Pull alpine through both `<harbor-host>/filter-hub/alpine:<alpine-tag>` and `<harbor-host>/filter-hub/library/alpine:<alpine-tag>`; both must succeed. Pull nginx through both forms and verify that both are denied.
8. Request `GET /v2/filter-hub/alpine/tags/list`. Verify the redirect to `/v2/filter-hub/library/alpine/tags/list` and a successful final response. Repeat with nginx; verify that the normalized request returns HTTP 404. Omitting `library/` must not bypass the filter.

### 11. Filter update audit logs and cleanup

1. As the system administrator, query `GET /api/v2.0/auditlog-exts` or open the new audit log view. Allow time for asynchronous audit records to appear and follow pagination when needed. Locate the project and metadata updates from the previous categories by project name, operator, and time. Verify operation `update`, resource type `project`, the correct project name and user, and `is_successful=true` for successful requests. Verify `is_successful=false` for rejected updates to existing projects.
2. Restore the original audit settings. Delete the test artifacts and projects, remove the two test users, and remove the registry endpoints created for this testcase. Remove only the upstream fixtures created for this test.
