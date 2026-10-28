**Description:** Verify configurable egress policies for HTTP webhook notifications: `allow_all`, `block_restricted`, and `public_only`; IPv4 CIDR allowlist exceptions; and compatibility with the deprecated private-network setting. With both changes applied, an unset or empty policy defaults to `allow_all` when the deprecated setting is also unset or empty.

**References:** [Webhook egress policies and allowlist, commit f6f5bac8d1](https://github.com/stonezdj/harbor/commit/f6f5bac8d1c05027bd483f7fb39831a7af497595) and [Default policy correction, PR #24126, commit cdec5400f1](https://github.com/goharbor/harbor/pull/24126/changes/cdec5400f1c512b83cbba50d6f8d6871f50ccc39).

**Steps**

Complete the categories in order. Use IPv4 addresses and DNS A records throughout this testcase. Expected results describe a build containing both commits; the second commit changes the original default from `block_restricted` to `allow_all`.

For every address-validation check below, use the UI or API methods in category 2, steps 2 and 3. API status codes correspond to the UI results in the table in step 3. After a rejected update, reopen or retrieve the policy and verify that its original target remains unchanged. Restricted addresses, including loopback and cloud metadata addresses, are validation-only targets and require no receivers.

Use notification type `http` for all policies. For delivery checks, enable only the notification policy being tested, select the artifact-push event, and push an image with a new tag. Check the webhook execution/task log and the receiver's request log. Successful delivery requires a received payload and a successful task; blocked delivery requires a policy error and no request reaching the receiver. Wait for retries to finish before changing settings.

When testing delivery to a target blocked by the current policy, first create and enable the notification under a policy that permits it, then apply the restrictive configuration without editing the notification.

Each YAML block below is a `harbor.yml` fragment. Replace the existing `webhook_egress` block with the shown block and update the shown `network` or `proxy` fields, preserving unrelated settings. Apply each fragment separately. `webhook_egress: {}` leaves the policy and allowlist unset; `network.allow_private_network_access: null` leaves the deprecated setting unset.

### 1. Environment and configuration

1. Install a Harbor v2.16.0-rc1, and log in as an administrator. Create project `webhook-egress` and prepare an image for repeated pushes.
2. Prepare controlled HTTP receivers with a public IPv4 address and a private IPv4 address such as `10.20.0.10`. Each receiver must log POST requests and return HTTP 200 for Harbor webhook payloads. Prepare DNS names resolving to these receivers. Use an actual reachable public address for delivery checks, rather than a documentation-only address.
3. Back up `harbor.yml` and any deployment environment overrides. Apply this fragment to exclude core and jobservice from proxy configuration, and clear any direct proxy environment overrides for those services. Confirm that both services can resolve the test DNS names, and that jobservice can reach each receiver before applying restrictive policies.

   ```bash
   cp harbor.yml harbor.yml.webhook-egress.bak
   ```

   ```yaml
   proxy:
     components:
       - registry
       - trivy
   ```

4. Apply YAML configuration changes by running `./prepare` and recreating the core and jobservice containers using the deployment's normal procedure. Check `common/config/core/env`, `common/config/jobservice/env`, and the effective container environments after each change. Both services must receive matching `HARBOR_WEBHOOK_EGRESS_POLICY` and `HARBOR_WEBHOOK_EGRESS_ALLOWLIST` values. A container restart alone does not load changed environment variables.
5. When a later step specifies a direct environment override, apply it to both services and recreate their containers without rerunning `./prepare`, which would replace the generated values. Restore these overrides before returning to YAML configuration tests.

### 2. Default behavior and explicit allow_all

1. Apply this fragment and remove corresponding environment overrides. Verify that the new policy and allowlist variables are empty and the deprecated `HARBOR_ALLOW_PRIVATE_NETWORK_ACCESS` variable is absent or empty in both services.

   ```yaml
   webhook_egress: {}
   network:
     allow_private_network_access: null
   ```

2. In Projects → `webhook-egress` → Webhooks, create notifications to the public and private receivers, testing one enabled policy at a time. Verify that saving and delivery succeed for both receivers through their IP addresses and DNS names.

   Log in as an administrator and open Projects → `webhook-egress` → Webhooks → **New Webhook**. Enter a unique name, select Notify Type **http**, Payload Format **Default**, and Event Type **Artifact pushed**. Enter the address under test in **Endpoint URL** and click **Add**. To verify an update, select an existing webhook, choose **Action → Edit**, change **Endpoint URL**, and click **Save**. For a blocked-update check, first create a webhook with an allowed address, then attempt to change it to the blocked address.

   For API creation, save this body as `webhook.json`, replacing the address with the controlled public or private receiver's URL. Use a unique policy name for each new policy:

   ```json
   {
     "name": "egress-address-check",
     "description": "Verify webhook destination policy",
     "enabled": true,
     "event_types": [
       "PUSH_ARTIFACT"
     ],
     "targets": [
       {
         "type": "http",
         "address": "http://<receiver-host>:<port>/hook",
         "auth_header": "",
         "skip_cert_verify": false,
         "payload_format": "Default"
       }
     ]
   }
   ```

   Replace `<harbor-host>` with Harbor's hostname; `--user admin` prompts for the administrator password:

   ```bash
   curl -i --user admin --request POST \
     'https://<harbor-host>/api/v2.0/projects/webhook-egress/webhook/policies' \
     --header 'Content-Type: application/json' \
     --data-binary @webhook.json
   ```

   To enable or deactivate an existing policy, send `PUT /api/v2.0/projects/webhook-egress/webhook/policies/<policy-id>` with this complete body, changing only `enabled` to `true` or `false`. Enable only the policy under test.

3. Using either method above, create and update validation-only policies targeting `http://169.254.169.254/hook`, `http://metadata.google.internal/hook`, and a syntactically valid hostname with no DNS record. Verify that target validation succeeds without requiring DNS resolution under the default policy. Delete these policies without triggering delivery.

   For API validation, use the following body with `enabled: false`, replacing `targets[0].address` with each address under test. Save it as `webhook.json` and use a unique name for each new policy. Submit it with the POST command in step 2:

   ```json
   {
     "name": "egress-address-check",
     "description": "Verify webhook destination policy",
     "enabled": false,
     "event_types": ["PUSH_ARTIFACT"],
     "targets": [
       {
         "type": "http",
         "address": "http://10.20.0.10/hook",
         "auth_header": "",
         "skip_cert_verify": false,
         "payload_format": "Default"
       }
     ]
   }
   ```

   For the update check, first create a policy with an allowed address and record `<policy-id>` from its `Location` header. Change only `targets[0].address` in the body and send:

   ```bash
   curl -i --user admin --request PUT \
     'https://<harbor-host>/api/v2.0/projects/webhook-egress/webhook/policies/<policy-id>' \
     --header 'Content-Type: application/json' \
     --data-binary @webhook.json
   ```

   Use `GET /api/v2.0/projects/webhook-egress/webhook/policies/<policy-id>` after a rejected update to confirm the original address is unchanged. Apply these expectations to address-validation checks in later categories:

   | Address result | UI expectation | API expectation |
   | --- | --- | --- |
   | Allowed on creation | Webhook is saved and appears in the list. | HTTP 201 with a `Location` header. |
   | Allowed on update | Changed address is saved. | HTTP 200. |
   | Blocked on creation or update | An error describes the blocked target; the new address is not saved. | HTTP 400 with a network-policy error. |

   The UI creates enabled webhooks. For validation-only checks, do not push artifacts; immediately deactivate a saved policy using **Action → Deactivate**, or delete it. For API deletion, send the following request with **no request body**; omit `--data` and `--data-binary`:

   ```bash
   curl -i --user admin --request DELETE \
     'https://<harbor-host>/api/v2.0/projects/webhook-egress/webhook/policies/<policy-id>'
   ```

4. Apply this empty-policy fragment, repeat steps 2 and 3, and verify the `allow_all` default.

   ```yaml
   webhook_egress:
     policy: ""
     allowlist: []
   network:
     allow_private_network_access: null
   ```

5. Apply this whitespace-policy fragment, repeat steps 2 and 3, and verify the same default.

   ```yaml
   webhook_egress:
     policy: "   "
     allowlist: []
   network:
     allow_private_network_access: null
   ```

6. Apply this explicit-policy fragment. Repeat steps 2 and 3 and verify the same results as the default.

   ```yaml
   webhook_egress:
     policy: allow_all
     allowlist: []
   network:
     allow_private_network_access: null
   ```

7. Apply this fragment and verify that the public receiver still works even though the allowlist contains only the private receiver's CIDR range; the allowlist must not become an exclusive destination list.

   ```yaml
   webhook_egress:
     policy: allow_all
     allowlist:
       - 10.20.0.0/16
   network:
     allow_private_network_access: null
   ```

8. Verify the runtime default with both policy variables entirely absent: remove `HARBOR_WEBHOOK_EGRESS_POLICY` and `HARBOR_ALLOW_PRIVATE_NETWORK_ACCESS` from `common/config/core/env` and `common/config/jobservice/env` (paths relative to the Harbor installation directory containing `harbor.yml`), leaving only the following egress-related entry. Recreate both containers without prepare, repeat steps 2 and 3, and verify `allow_all` behavior.

   ```dotenv
   HARBOR_WEBHOOK_EGRESS_ALLOWLIST=
   ```

### 3. Explicit block_restricted

1. Apply this fragment. Verify that saving and delivery to the public and private receivers succeed, including their DNS names.

   ```yaml
   webhook_egress:
     policy: block_restricted
     allowlist: []
   network:
     allow_private_network_access: null
   ```

2. Create and update validation-only policies using each of these target hosts: `127.0.0.1`, `localhost`, `test.localhost`, `169.254.1.1`, `0.0.0.0`, `224.0.0.1`, and `255.255.255.255`. Use valid HTTP URLs and verify that every attempt is rejected as restricted.
3. Repeat the rejection checks for metadata/control targets `169.254.169.254`, `100.100.100.200`, `168.63.129.16`, `metadata.google.internal`, and `metadata.azure.com`. Verify that subdomains of the two metadata hostnames are also rejected.
4. Configure a test hostname resolving only to the restricted address `169.254.1.1`, then one with both a permitted address and that restricted address. Verify that creation and update of validation-only policies are rejected in both cases; every resolved address must be permitted. Verify that a non-allowlisted hostname with no DNS answer is rejected with a resolution error.
5. Check core logs for the validation failures: each denial must identify the policy and target and mention `HARBOR_WEBHOOK_EGRESS_ALLOWLIST` or `HARBOR_WEBHOOK_EGRESS_POLICY` as an operator setting. API errors must describe the blocked target without exposing those configuration-variable names.

### 4. Explicit public_only

1. Apply this fragment. Verify that saving and delivery to the public receiver succeed through both its IP address and DNS name.

   ```yaml
   webhook_egress:
     policy: public_only
     allowlist: []
   network:
     allow_private_network_access: null
   ```

2. Verify that creation and update are rejected for private addresses in `10.0.0.0/8`, `172.16.0.0/12`, and `192.168.0.0/16`, including the private receiver and its DNS name.
3. Verify that representative non-public addresses `100.64.0.1`, `198.18.0.1`, and `192.0.2.1` are rejected. Repeat the restricted-target checks from category 3 and verify that they remain rejected.
4. Configure a hostname with two A records, one pointing to a public address and the other to a private address. Verify that creation and update fail even when one address is public.
5. Apply the fragment in category 3, step 1, and create an enabled private-receiver notification. Reapply the `public_only` fragment in this category's step 1 and trigger a push without editing the notification. Verify that jobservice rejects the delivery with a private-network error and the receiver logs no request. Verify that the jobservice service log identifies the policy and target and mentions `HARBOR_WEBHOOK_EGRESS_ALLOWLIST` or `HARBOR_WEBHOOK_EGRESS_POLICY` as an operator setting.

### 5. CIDR allowlist

1. Apply this CIDR allowlist fragment and verify that both service environments contain `HARBOR_WEBHOOK_EGRESS_ALLOWLIST=10.20.0.0/16`.

   ```yaml
   webhook_egress:
     policy: public_only
     allowlist:
       - 10.20.0.0/16
   network:
     allow_private_network_access: null
   ```

2. Create an enabled notification targeting the private receiver by its IP address `10.20.0.10`, which is within the allowed range. Verify that saving and delivery succeed.
3. Using either method above, attempt to create a validation-only policy targeting `http://10.21.0.10/hook`, which is outside the allowed range, and attempt to update an existing policy to that address. Verify that both attempts are rejected: an error in the UI or HTTP 400 through the API. No receiver is required for this validation check.

   Start with the complete body from category 2, step 3. The allowed target is `http://10.20.0.10/hook`; change `targets[0].address` to `http://10.21.0.10/hook` for both the rejected POST and PUT. Keep `enabled: false` and use a new name for POST. Submit the bodies to `POST /api/v2.0/projects/webhook-egress/webhook/policies` and `PUT /api/v2.0/projects/webhook-egress/webhook/policies/<policy-id>`, respectively.

### 6. Deprecated setting and precedence

1. Apply this fragment. Verify the prepare deprecation warning and `HARBOR_ALLOW_PRIVATE_NETWORK_ACCESS=true` in both environments. Verify that public/private saving and delivery succeed and that the validation-only checks from category 2, step 3 succeed, preserving `allow_all` behavior.

   ```yaml
   webhook_egress: {}
   network:
     allow_private_network_access: true
   ```

2. Apply this fragment. Verify that the literal string `false` is present in both environments. Verify public delivery succeeds, private and restricted targets fail validation, and an existing private notification fails delivery. This must behave as `public_only`.

   ```yaml
   webhook_egress: {}
   network:
     allow_private_network_access: false
   ```

3. Apply this fragment. Verify that private-receiver validation and retained notification delivery remain blocked: the explicit policy must take precedence in both core and jobservice.

   ```yaml
   webhook_egress:
     policy: public_only
     allowlist: []
   network:
     allow_private_network_access: true
   ```

4. Apply this fragment. Verify that saving and delivery to the private receiver succeed despite the deprecated value being `false`.

   ```yaml
   webhook_egress:
     policy: allow_all
     allowlist: []
   network:
     allow_private_network_access: false
   ```

5. Apply this empty-policy fragment and verify `public_only` behavior: public delivery succeeds and private targets are blocked.

   ```yaml
   webhook_egress:
     policy: ""
     allowlist: []
   network:
     allow_private_network_access: false
   ```

6. Apply this whitespace-policy fragment and verify the same `public_only` behavior.

   ```yaml
   webhook_egress:
     policy: "   "
     allowlist: []
   network:
     allow_private_network_access: false
   ```

7. Reapply the fragments in category 2, steps 4 and 5, one at a time. Verify that each returns to `allow_all` behavior when the deprecated setting is unset.
8. Apply this fragment. Verify that saving and delivery to `10.20.0.10` succeed while validation rejects `10.21.0.10`. Check service logs for a deprecation warning when the legacy setting is used.

   ```yaml
   webhook_egress:
     allowlist:
       - 10.20.0.0/16
   network:
     allow_private_network_access: false
   ```

### 7. Configuration normalization and invalid values

1. Apply this fragment and run `./prepare`. Verify that both generated environments contain `block_restricted`. Recreate the containers and verify that private-receiver delivery succeeds and validation rejects a validation-only policy targeting `http://169.254.169.254/hook`.

   ```yaml
   webhook_egress:
     policy: " Block_Restricted "
     allowlist: []
   network:
     allow_private_network_access: null
   ```

2. Apply this fragment and run `./prepare`. Verify that prepare fails with an error listing `allow_all`, `block_restricted`, and `public_only` as valid values. Restore the valid fragment from category 3, step 1, and rerun prepare before continuing.

   ```yaml
   webhook_egress:
     policy: allow-all
     allowlist: []
   network:
     allow_private_network_access: null
   ```

3. For this runtime check, replace the matching entries in both `common/config/core/env` and `common/config/jobservice/env` with the following environment-file snippet. Recreate the containers without prepare. Verify that the invalid explicit value produces a warning and falls back to `public_only`: the public receiver works, private/restricted targets fail validation, and retained private notifications fail delivery despite the legacy value being `true`.

   ```dotenv
   HARBOR_WEBHOOK_EGRESS_POLICY=allow-all
   HARBOR_WEBHOOK_EGRESS_ALLOWLIST=
   HARBOR_ALLOW_PRIVATE_NETWORK_ACCESS=true
   ```

4. Replace the same entries in both environment files with this snippet and recreate the containers without prepare. Verify the invalid-legacy warning and the same `public_only` behavior. Trigger several requests with unchanged configuration and verify that the parsing warning does not repeat for every request within the same process; individual blocked-target warnings may repeat.

   ```dotenv
   HARBOR_WEBHOOK_EGRESS_POLICY=
   HARBOR_WEBHOOK_EGRESS_ALLOWLIST=
   HARBOR_ALLOW_PRIVATE_NETWORK_ACCESS=flase
   ```

5. Reapply the valid fragment from category 3, step 1, and regenerate configuration with prepare to replace the environment-file changes. Remove any deployment environment overrides and recreate the containers. Verify that both services have matching effective settings.

### 8. Persistence and cleanup

1. Reapply the CIDR fragment from category 5, step 1, verify permitted and blocked deliveries, and restart Harbor. Verify that both services retain the configured policy and allowlist and that the same delivery results hold after restart.
2. Restore the backed-up `harbor.yml` with the command below, then restore the original environment overrides and DNS records. Regenerate configuration and recreate services. Delete test notifications, images, and project `webhook-egress`, and stop the receivers.

   ```bash
   cp harbor.yml.webhook-egress.bak harbor.yml
   ```

   Delete each test notification with this request, which has no body:

   ```bash
   curl -i --user admin --request DELETE \
     'https://<harbor-host>/api/v2.0/projects/webhook-egress/webhook/policies/<policy-id>'
   ```

3. Record failures with the Harbor build, effective policy and allowlist, target URL, API response, execution/task log, service log, and receiver observations.
