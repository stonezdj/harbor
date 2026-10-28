**Description:** Verify that proxy cache project creation defaults to a 7-day retention period and that a custom retention period creates a working tag retention policy.

**Reference:** [PR #23863](https://github.com/goharbor/harbor/pull/23863).

**Steps**

1. Install a Harbor build containing PR #23863 and log in as an administrator.
2. Go to Administration → Registries, add an upstream registry containing a small tagged test image, and verify that the connection succeeds. Use the same endpoint for all projects below.
3. Go to Projects → New Project. Verify that **Retention period (days)** appears after enabling **Proxy Cache** and defaults to **7**. Select the endpoint and create project `cache-default` without changing the retention period.
4. Open `cache-default` → **Tag Retention**. Verify that the generated rule retains artifacts pulled within the last **7 days**, matches all repositories and tags (`**`), and runs **daily**.
5. Open New Project again and enable Proxy Cache. Verify that the retention period defaults to **7**, change it to **30**, and create project `cache-custom` with the same endpoint.
6. Open `cache-custom` → **Tag Retention**. Verify that the rule uses **30 days**, matches all repositories and tags, and runs daily. Refresh the page and verify that the value remains **30**; the rule in `cache-default` must still use **7**.
7. Run `docker login <harbor-host>`, then pull the test image through each project using `docker pull <harbor-host>/<proxy-project>/<upstream-image>:<tag>`, replacing `<proxy-project>` with `cache-default` and `cache-custom` and including any upstream namespace in `<upstream-image>`. Wait until the image appears in each project. In each project's Tag Retention tab, click **RUN NOW**, confirm **RUN**, and wait for successful completion. Verify that the recently pulled image and tag are retained.
8. Delete the test images and projects, then remove the registry endpoint created for this testcase.
