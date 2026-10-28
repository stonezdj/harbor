**Description:** Verify that Harbor records replication task failure details, aggregates them into an execution failure summary, and displays the details in API responses and portal tooltips.

**Reference:** [PR #24028](https://github.com/goharbor/harbor/pull/24028).

**Steps**

Complete the categories in order. Use administrator authentication for API requests. Execution summaries are returned in `status_text`; task details are returned in `status_message`. Wait for automatic retries and execution status updates to finish before checking final results, except when explicitly testing retry behavior. Follow API pagination when counting tasks.

### 1. Environment and test data

1. Install a Harbor build containing PR #24028 and prepare a second Harbor instance as the remote registry. Log in to both instances as an administrator. Record the tested build versions.
2. On the remote registry, create project `error-source` and push a single-platform image with one tag into `error-source/image-a`. Use an image with compressed layers larger than 1 MiB. On the Harbor instance under test, create an empty project named `error-destination` with a storage quota of 1 MiB.
3. Go to Administration → Registries, add the remote Harbor endpoint with valid credentials, and verify that its connection test succeeds. Go to Administration → Replications and create a manual pull rule matching only `error-source/image-a` and its tag. Set the destination namespace and flattening so the image maps to `error-destination/image-a`.

### 2. Single task failure and API details

1. Select the pull rule and click **Replicate**. Wait for the execution to finish with status **Failed**. Record its execution ID as `<execution-id>` and the rule ID as `<policy-id>`.
2. Request `GET /api/v2.0/replication/executions/<execution-id>/tasks`. Verify HTTP 200, one failed task, and a nonempty `status_message` identifying the destination quota failure. Record the task ID as `<task-id>`.
3. Request `GET /api/v2.0/replication/executions/<execution-id>/tasks/<task-id>/log`. Verify that the full log remains accessible. Compare it with `status_message`: the message must retain the job's returned error and the last three nonempty log lines available when the failure was recorded, subject to the message length limit. The quota failure must be understandable from the task response alone.
4. Request `GET /api/v2.0/replication/executions/<execution-id>`. Verify `status="Failed"`, `total=1`, `failed=1`, `succeed=0`, and `in_progress=0`. Verify that `status_text` contains the failed task's message, without requiring a separate log request to discover the cause.
5. Request `GET /api/v2.0/replication/executions?policy_id=<policy-id>`. Locate the same execution and verify that its status, counters, and `status_text` match the execution detail response.

### 3. Failure details in the portal

1. Open the replication rule's execution list. Hover over the information icon beside the failed execution status. Verify that its tooltip displays the execution API's `status_text`.
2. Open the execution details. Verify that the information icon beside the execution's failed status displays the same summary. Hover over the information icon beside the failed task's status and verify that it displays that task's `status_message`.
3. Verify that multiline messages preserve line breaks, long lines wrap within the tooltip, and messages taller than the tooltip can be scrolled. Open the task log and verify that the existing log view still works. Refresh the page and verify that the failure details remain available.

### 4. Different failures and successful tasks in one execution

1. Create projects `summary-ok`, `summary-quota`, and `summary-immutable` on both Harbor instances. On the instance under test, push one single-platform image with one tag into one repository in each project. Give each image a different digest from any existing destination image. On the remote instance, leave `summary-ok` unrestricted, set the empty `summary-quota` project to 1 MiB, and ensure its source image exceeds that quota.
2. In remote project `summary-immutable`, push a different image using the same repository and tag as its source image, then create a tag immutability rule covering that tag. Create a manual push replication rule on the instance under test matching exactly these three source repositories. Leave the destination namespace empty to preserve project names, and enable **Override** so the immutable tag replacement is attempted.
3. Run the rule and wait for completion. Verify that the `summary-ok` task succeeds, the `summary-quota` task fails with a quota error, and the `summary-immutable` task fails with an immutability error. Verify `total=3`, `failed=2`, `succeed=1`, and execution status **Failed**.
4. Verify that `status_text` reports two failed tasks and includes both distinct failure messages. Single-line messages should be listed separately; multiline messages should retain their line breaks and be separated by `---`. The successful task must contribute no error message and show no failure-details icon when its `status_message` is empty or omitted.
5. Verify the execution summary and individual task messages in the portal. Refresh the API responses and page repeatedly after completion; the summary must remain stable without accumulating duplicate copies of the same errors.

### 5. More than 50 failed tasks

1. Prepare 60 separate repositories in the remote `error-source` project, each containing one tagged single-platform image that cannot fit within the destination quota. Update the pull rule to match exactly these 60 repositories and trigger replication.
2. After completion, retrieve every page of tasks. Verify 60 failed tasks with failure details and execution counters `total=60` and `failed=60`.
3. Verify that the execution summary reports all 60 failures even though aggregation samples at most 50 failed tasks. For multiple distinct sampled messages, verify wording indicating error samples or **at least** the sampled number of distinct errors. Verify that the decoded `status_text` is at most 4096 UTF-8 bytes and ends with `...` if the aggregate is truncated. Full per-task logs must remain accessible.
4. Open the summary tooltip. Verify that long content wraps and scrolls without expanding the execution table beyond the page. Confirm that the execution list and detail API return the same summary.

### 6. Retry, recovery, and scheduled execution

1. Restore the pull rule to match only `error-source/image-a`. Start a new execution with the destination quota still too small. Monitor its task response until the first failure message appears, then increase the destination quota before the next automatic job retry starts.
2. Follow the same task ID through the retry. Verify that the previous `status_message` is cleared when the new attempt starts and remains empty or omitted when the task succeeds. Once the execution returns to **InProgress**, verify that its previous failure summary is cleared; after successful completion, verify `status="Succeed"`, `failed=0`, and an empty `status_text`.
3. Trigger a fresh manual execution with sufficient quota. Verify successful completion and no failure-details icons for empty messages. Reopen the completed failed executions from earlier categories and verify that their historical failure details remain available.
4. Create a separate empty destination project with a 1 MiB quota and a scheduled pull rule for `error-source/image-a`. Wait for a scheduled execution to fail. Verify `trigger="scheduled"` and the same task messages, execution summary, and tooltips as for the manual quota failure. Disable the schedule afterward.

### 7. Persistence and cleanup

1. After all executions finish, restart Harbor using the deployment's normal procedure. Retrieve a previously failed execution and its tasks, then reopen their tooltips. Verify that the stored failure messages and counters are preserved.
2. Disable or delete the test replication rules. Remove the test tag immutability rule, registry endpoint, images, and projects created by this testcase. Record any failures with the Harbor build, execution ID, task ID, API responses, and corresponding job log.
