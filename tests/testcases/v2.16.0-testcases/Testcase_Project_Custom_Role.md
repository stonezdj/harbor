**Description:** Verify project custom roles. System administrators can create reusable roles with selected project permissions, and project administrators can assign them to users or groups independently in each project. Verify role management, permission enforcement, immediate permission changes, privilege escalation prevention, audit logging, and compatibility with existing built-in roles and robot accounts.

**Reference:** [Custom Project Roles proposal, goharbor/community#276](https://github.com/goharbor/community/pull/276). Built-in roles are expected to be immutable, following the proposal's Key Design Decisions and the [review conclusion](https://github.com/goharbor/community/pull/276#discussion_r3471513456), although the Open Issues section still describes modification. API details and validation checks below also use the current repository's [API specification](../../../api/v2.0/swagger.yaml).

**Steps**

Complete the categories in order. Substep numbering restarts within each category.

1. **Environment setup**

   1. Install a Harbor v2.16.0-rc1. Use db auth for the individual-user tests. Enable audit logging for the operations under test.
   2. Log in as the system administrator. Create private projects `custom-role-a` (A), `custom-role-b` (B), and `custom-role-c` (C), with two repositories and tagged images in each project. Create ordinary users `project-admin`, `devops-user`, `auditor-user`, `release-user`, `delegate-user`, `target-user`, `api-user`, and `outsider`. Assign `project-admin` the built-in Project Admin role in projects A and B only. Keep the other users free of project memberships initially. Use separate browser profiles and Docker configurations for different users.

2. **Role creation and validation**

   1. Go to Administration → Roles. Verify that Project Admin, Maintainer, Developer, Guest, and Limited Guest are listed as built-in roles. Open their details and record their permissions. Verify that `GET /api/v2.0/roles` and `GET /api/v2.0/roles/{role_id}` return their names, IDs, built-in status, and permissions consistently with the UI.
   2. As the system administrator, open the role creation dialog. Verify that the available permissions match the project-role catalog returned by `GET /api/v2.0/permissions`. System administration permissions must not be available as project-role grants.
   3. Create the following custom roles, each with a description. Select the read/list permissions needed to open the relevant project pages as well as the permissions for the actions listed below. Leave all other permissions unselected. Verify that each role appears in the Roles list with its exact name, description, and selected permissions.

      | Role | Permissions to select | Actions that must remain denied |
      | --- | --- | --- |
      | `Project Reader` | Read/list project details, repositories, artifacts, and tags; pull images | Push, artifact or repository deletion, scans, robot management, member management, and project configuration changes |
      | `DevOps Engineer` | All `Project Reader` permissions; push images; create, read/list, update, and delete project robot accounts | Artifact or repository deletion, member management, and project configuration changes |
      | `Security Auditor` | All `Project Reader` permissions; trigger scans and read scan results | Push, artifact or repository deletion, robot management, member management, and project configuration changes |
      | `Release Manager` | All `Project Reader` permissions; push and delete artifacts; manage tags and tag immutability rules | Repository deletion, robot management, member management, and other project configuration changes |
      | `Member Delegate` | All `Project Reader` permissions; create, read/list, update, and delete project members | Push, scans, robot management, and project configuration changes |

   4. Refresh the page and retrieve each role by ID through the API. Verify that the custom roles have `is_builtin=false`, retain their permission selections, and record the creating administrator and creation time. Create an additional unassigned role through `POST /api/v2.0/roles`; verify HTTP 201 and that the returned ID and Location identify the same role in the UI.

      Use `Content-Type: application/json` for JSON requests and authenticate as the actor named in each step. Replace `<harbor-host>` and path placeholders with actual values. Example IDs such as `101` are illustrative; replace them with actual IDs and keep them as JSON numbers. For unauthenticated checks, omit authentication entirely.

      Save this body as `role-create.json`. Role definitions use `kind: "project-role"` and namespace `"*"`; choose resource/action pairs from the `role` array returned by `GET /api/v2.0/permissions`. Basic project visibility comes from membership, so do not add an empty-resource permission.

      ```json
      {
        "name": "API Test Reader",
        "description": "Unassigned role for API lifecycle checks",
        "permissions": [
          {
            "kind": "project-role",
            "namespace": "*",
            "access": [
              {"resource": "repository", "action": "list", "effect": "allow"},
              {"resource": "repository", "action": "read", "effect": "allow"},
              {"resource": "repository", "action": "pull", "effect": "allow"},
              {"resource": "artifact", "action": "list", "effect": "allow"},
              {"resource": "artifact", "action": "read", "effect": "allow"},
              {"resource": "tag", "action": "list", "effect": "allow"}
            ]
          }
        ]
      }
      ```

      Submit the request below; `--user admin` prompts for the password. Use the same curl pattern for later requests, changing the actor, method, endpoint, and payload file as specified in the step.

      ```bash
      curl -i --user admin --request POST \
        'https://<harbor-host>/api/v2.0/roles' \
        --header 'Content-Type: application/json' \
        --data-binary @role-create.json
      ```

   5. As the system administrator, test role creation with an empty name, a duplicate existing name, a name containing `+`, a name longer than 255 ASCII characters, and no selected permissions. Verify that invalid input is rejected without creating a role. Submit equivalent API requests, plus requests containing an unknown resource/action or system-level permissions; verify validation errors and no partial role creation. Verify that a valid unique name and valid project permissions still succeed afterward.

      For each `POST /api/v2.0/roles`, start from `role-create.json`, use a fresh name unless testing the name itself, and apply only the indicated change:

      | Validation case | Change to the sample body |
      | --- | --- |
      | Empty name | Set `name` to `""`. |
      | Duplicate name | Keep `name: "API Test Reader"` after successfully creating that role. |
      | Reserved character | Set `name` to `"API+Reader"`. |
      | Name longer than 255 ASCII characters | Generate a body with `jq '.name = ("a" * 256)' role-create.json > role-invalid.json`. |
      | No permissions | Set `permissions` to `[]`. Also test `permissions[0].access: []` separately. |
      | Unknown resource | Set `permissions[0].access[0].resource` to `"unknown-resource"`. |
      | Unknown action | Set `permissions[0].access[0].action` to `"unknown-action"`. |
      | System resource in a project role | Replace `permissions[0].access` with `[{"resource":"user","action":"create","effect":"allow"}]`. |
      | System permission scope | Replace `permissions` with `[{"kind":"system","namespace":"/","access":[{"resource":"user","action":"create","effect":"allow"}]}]`. |

3. **Role management access and built-in role protection**

   1. Log in as `project-admin`, then as `outsider`. Verify that neither user can create, edit, or delete role definitions through the UI. Send a valid role creation request and valid update/deletion requests targeting the unassigned custom role from category 2, step 4 directly to the role API as each user; verify that they are forbidden and no role changes occur. Repeat without authentication and verify that the requests are rejected. Authenticated role reads must still support the project member role selector.

      Use these requests as each tested user, then repeat without authentication. Generate the two bodies from category 2, step 4's sample:

      ```bash
      jq '.name = "API Unauthorized Reader"' role-create.json > role-denied-create.json
      jq '.description = "Unauthorized update attempt"' role-create.json > role-denied-update.json
      ```

      | Method and endpoint | Request body |
      | --- | --- |
      | `POST /api/v2.0/roles` | `role-denied-create.json`; use an unused name. |
      | `PUT /api/v2.0/roles/<role-id>` | `role-denied-update.json`; target the unassigned role. |
      | `DELETE /api/v2.0/roles/<role-id>` | No body; omit `--data` and `--data-binary`. |

   2. As the system administrator, try to edit and delete each built-in role through the UI and direct API requests. Verify that these operations are blocked, including a request attempting to clear its built-in flag. Retrieve the roles again and compare their IDs, names, and permissions with category 2, step 1; they must remain unchanged.

      Save each built-in role's GET response for comparison afterward. Generate a PUT body from `role-create.json`, replacing `<builtin-role-name>` with the exact recorded name, and submit it to `PUT /api/v2.0/roles/<builtin-role-id>`. The sample permissions are valid role-catalog entries, allowing the request to reach the built-in protection check. Repeat with the flag-clearing body:

      ```bash
      jq --arg name '<builtin-role-name>' \
        '.name = $name | .description = "Attempted built-in role update"' \
        role-create.json > builtin-role-update.json
      jq '.is_builtin = false' builtin-role-update.json > builtin-role-clear-flag.json
      ```

      Use `DELETE /api/v2.0/roles/<builtin-role-id>` for the deletion check, with no request body.

4. **User assignment and permission enforcement**

   1. Log in as `project-admin` and open project A → Members → Add User. Verify that the role selector contains both built-in and custom roles. Assign `DevOps Engineer` to `devops-user`, `Security Auditor` to `auditor-user`, `Release Manager` to `release-user`, and `Member Delegate` to `delegate-user`. Verify that the saved member list displays the correct role names.
   2. In project B, assign `DevOps Engineer` to both `devops-user` and `auditor-user`. Verify that the same role definition is reusable across projects and that `auditor-user` retains `Security Auditor` in project A. Do not assign these users to project C.
   3. Log in as `devops-user`. Verify that projects A and B are accessible. Use Docker to pull existing images and push new tags into both repositories in project A and a repository in project B. Verify that all operations succeed: the permissions apply throughout each assigned project.
   4. As `devops-user`, attempt to delete an artifact, delete a repository, add or remove a project member, change a member's role, and change project configuration. Verify that the UI prevents these operations and that equivalent direct API requests are denied without changing any resources. Also verify that the user cannot create Harbor users or change system configuration.

      Send these requests as `devops-user`, using existing disposable resources. Replace `101` with the actual `Project Reader` ID and `<member-id>` with a project membership ID, not a user ID. For artifact endpoints, use an existing repository and artifact digest or tag; encode path values as required by the API.

      | Method and endpoint | Sample JSON body |
      | --- | --- |
      | `DELETE /api/v2.0/projects/custom-role-a/repositories/<repository>/artifacts/<reference>` | No body. |
      | `DELETE /api/v2.0/projects/custom-role-a/repositories/<repository>` | No body. |
      | `POST /api/v2.0/projects/custom-role-a/members` | `{"role_id":101,"member_user":{"username":"target-user"}}` |
      | `PUT /api/v2.0/projects/custom-role-a/members/<member-id>` | `{"role_id":101}` |
      | `DELETE /api/v2.0/projects/custom-role-a/members/<member-id>` | No body. |
      | `PUT /api/v2.0/projects/custom-role-a` | `{"metadata":{"auto_scan":"true"}}` |
      | `PUT /api/v2.0/configurations` | `{"read_only":true}` |

      For `POST /api/v2.0/users`, use this body with an unused username. Send Basic authentication without browser session cookies:

      ```json
      {
        "username": "custom-role-denied-user",
        "email": "custom-role-denied-user@example.com",
        "realname": "Custom Role Denied User",
        "password": "Test-Only-Password-123!",
        "comment": "Must not be created by a project custom role"
      }
      ```

   5. Log in as `auditor-user` and open project A. Verify that repositories and artifacts can be viewed, images can be pulled, and a vulnerability scan can be started and its result read. Verify that push, artifact deletion, robot creation, member changes, and project configuration changes are denied through the applicable UI, Docker, and API operations.

      For the allowed scan, send `POST /api/v2.0/projects/custom-role-a/repositories/<repository>/artifacts/<reference>/scan` with:

      ```json
      {"scan_type":"vulnerability"}
      ```

      For denied robot creation, send `POST /api/v2.0/robots` with the following body, saved as `robot-create.json`. Use a unique robot name for each creation attempt. Repeat the artifact deletion, member, and project configuration requests from step 4 as `auditor-user`.

      ```json
      {
        "name": "custom-role-pull-check",
        "description": "Project robot for custom role permission checks",
        "level": "project",
        "duration": 1,
        "disable": false,
        "permissions": [
          {
            "kind": "project",
            "namespace": "custom-role-a",
            "access": [
              {"resource": "repository", "action": "pull", "effect": "allow"}
            ]
          }
        ]
      }
      ```

   6. As `auditor-user`, open project B and verify that push and project robot creation succeed under `DevOps Engineer`. Repeat those actions in project A and verify that they remain denied. Verify that this user and `devops-user` cannot access project C through its direct URL, project API, or Docker pull/push, and that `outsider` cannot access any of the three private projects.

      For robot creation in B, send `POST /api/v2.0/robots` with `robot-project-b.json` generated below. Repeat with `robot-project-a.json` for A:

      ```bash
      jq '.name = "auditor-project-b-check" | .permissions[0].namespace = "custom-role-b"' \
        robot-create.json > robot-project-b.json
      jq '.name = "auditor-project-a-check" | .permissions[0].namespace = "custom-role-a"' \
        robot-create.json > robot-project-a.json
      ```

   7. Log in as `release-user`. In project A, push a disposable image, add and delete a tag, and delete a disposable artifact; verify success. Create, update, and delete an immutability rule matching a separate set of test tags; verify that matching tags are protected while the rule is active. Attempt repository deletion and member management through the UI and API; verify denial.

      Use these requests as `release-user`:

      | Method and endpoint | Sample JSON body |
      | --- | --- |
      | `POST /api/v2.0/projects/custom-role-a/repositories/<repository>/artifacts/<reference>/tags` | `{"name":"release-check"}` |
      | `DELETE /api/v2.0/projects/custom-role-a/repositories/<repository>/artifacts/<reference>/tags/release-check` | No body. |
      | `DELETE /api/v2.0/projects/custom-role-a/repositories/<repository>/artifacts/<reference>` | No body. |
      | `DELETE /api/v2.0/projects/custom-role-a/repositories/<repository>` | No body; expect denial. |
      | `POST /api/v2.0/projects/custom-role-a/members` | `{"role_id":101,"member_user":{"username":"target-user"}}`; use the actual `Project Reader` ID and expect denial. |

      For `POST /api/v2.0/projects/custom-role-a/immutabletagrules`, use:

      ```json
      {
        "priority": 0,
        "disabled": false,
        "action": "immutable",
        "template": "immutable_template",
        "tag_selectors": [
          {"kind": "doublestar", "decoration": "matches", "pattern": "immutable-check-*"}
        ],
        "scope_selectors": {
          "repository": [
            {"kind": "doublestar", "decoration": "repoMatches", "pattern": "**"}
          ]
        }
      }
      ```

      For `PUT /api/v2.0/projects/custom-role-a/immutabletagrules/<rule-id>`, reuse the complete body and change the tag pattern to `immutable-check-updated-*`. Delete it with `DELETE /api/v2.0/projects/custom-role-a/immutabletagrules/<rule-id>` and no body.

   8. As `project-admin`, use the existing project member APIs to add `api-user` to project A with a custom role ID, change it to a built-in role, and change it back. Verify successful responses and matching UI role names. Try a nonexistent role ID; verify rejection with no invalid membership saved. Remove this project membership afterward.

      For `POST /api/v2.0/projects/custom-role-a/members`, replace `101` with the selected custom role ID:

      ```json
      {
        "role_id": 101,
        "member_user": {"username": "api-user"}
      }
      ```

      For `PUT /api/v2.0/projects/custom-role-a/members/<member-id>`, change the membership to built-in Developer with this body. Use the membership ID returned by the member list, not the user's ID:

      ```json
      {
        "role_id": 2
      }
      ```

      Replace `2` with the custom role ID to change it back. For invalid-role checks, use an ID confirmed not to exist in the POST and PUT bodies. Remove the membership with `DELETE /api/v2.0/projects/custom-role-a/members/<member-id>` and no body.

5. **Robot permissions and privilege escalation prevention**

   1. As `devops-user`, create a project robot account in project A with pull permission, then update it to add push permission. Verify that these operations succeed, the robot can perform its granted operations, and it cannot delete artifacts or manage project members. Create and delete a second disposable robot account to verify the granted robot deletion permission.

      Send `POST /api/v2.0/robots` with `robot-create.json` from category 4, step 5, using a fresh name. For `PUT /api/v2.0/robots/<robot-id>`, save the robot's GET response as `robot-current.json` and generate this update body. Preserve the exact returned name, including its prefix, and the existing level and project namespace:

      ```bash
      jq '{name, description, level, duration, disable, permissions} |
        .permissions[0].access = [
          {"resource":"repository","action":"pull","effect":"allow"},
          {"resource":"repository","action":"push","effect":"allow"}
        ]' robot-current.json > robot-update.json
      ```

      Submit `robot-update.json` as the PUT body. For the disposable second robot, repeat POST with another unused name, then send `DELETE /api/v2.0/robots/<robot-id>` with no body. Test the pull-only and pull/push robot's denials using category 4, step 4's artifact deletion and member requests with the robot's credentials.

   2. As `devops-user`, try to create or update a project robot with repository deletion or other permissions the user does not hold. Also try to include project C or system-level permissions in a direct API request. Verify that excessive grants are unavailable in the UI or rejected on submission, that the API rejects them, and that a failed update leaves the existing robot permissions unchanged.

      Start from a fresh copy of `robot-create.json` for `POST /api/v2.0/robots`, or `robot-update.json` for `PUT /api/v2.0/robots/<robot-id>`, and apply one change per request:

      | Excessive grant | Change to the sample body |
      | --- | --- |
      | Repository deletion | Append `{"resource":"repository","action":"delete","effect":"allow"}` to `permissions[0].access`. |
      | Access to project C | Append `{"kind":"project","namespace":"custom-role-c","access":[{"resource":"repository","action":"pull","effect":"allow"}]}` to `permissions`. |
      | System permission | Append `{"kind":"system","namespace":"/","access":[{"resource":"user","action":"create","effect":"allow"}]}` to `permissions`. Also test POST with `level: "system"` and this system permission alone to verify that `devops-user` cannot create system robots. |

   3. Log in as `delegate-user`. Add `target-user` to project A with `Project Reader`, then change it to `Member Delegate` and back to `Project Reader`. Verify success because the assigned permissions are a subset of the delegate's own permissions. Attempt to assign `DevOps Engineer` or built-in Project Admin to a new member and to `target-user`, using both the UI and direct member API requests. Verify that the excessive assignments are rejected and existing assignments remain unchanged.

      Use these bodies as `delegate-user`. Replace `101` with `Project Reader`'s actual ID, then with `Member Delegate`'s ID for the allowed change. For excessive assignments, substitute the actual `DevOps Engineer` ID or built-in Project Admin ID `1`. Use an unassigned test user for POST and the existing target membership for PUT.

      | Method and endpoint | Sample JSON body |
      | --- | --- |
      | `POST /api/v2.0/projects/custom-role-a/members` | `{"role_id":101,"member_user":{"username":"target-user"}}` |
      | `PUT /api/v2.0/projects/custom-role-a/members/<member-id>` | `{"role_id":101}` |

6. **Role updates and membership revocation**

   1. Keep `devops-user` logged in. As the system administrator, edit the `DevOps Engineer` description and remove only robot creation permission. Verify that the updated description, permission set, modifying administrator, and modification time are returned by the role API.

      Save the current `DevOps Engineer` GET response as `devops-role-original.json`. Generate the following body and send it to `PUT /api/v2.0/roles/<devops-role-id>`, retaining all other permissions:

      ```bash
      jq '.description = "Robot creation temporarily revoked" |
        .permissions[].access |= map(select(.resource != "robot" or .action != "create"))' \
        devops-role-original.json > devops-role-no-robot-create.json
      ```

   2. Without logging out `devops-user`, immediately submit new robot creation requests in both projects A and B. Verify that the next API request is denied in both projects, while still-granted pull and robot read/list operations continue to succeed. Refresh the UI and verify that its available actions match the updated permissions. Restore robot creation permission as the system administrator and verify that the user's next creation request succeeds without another login. Repeat across Harbor core replicas when testing an HA deployment.

      Send `POST /api/v2.0/robots` with the body from category 4, step 5, using a fresh name for every attempt and setting `permissions[0].namespace` to `custom-role-a` or `custom-role-b`. Restore the role with `PUT /api/v2.0/roles/<devops-role-id>` using `devops-role-original.json`, then repeat the robot creation requests.

   3. Remove push permission from `DevOps Engineer`. Using a fresh Docker client process and newly obtained registry authorization, verify that push fails in both assigned projects while pull succeeds. Restore push permission and verify success with fresh authorization. Test next-request revocation through the Harbor API separately from already-issued registry bearer tokens, whose validity follows the existing token expiration behavior.

      Generate this body from the original role and send it to `PUT /api/v2.0/roles/<devops-role-id>`. Restore by sending `devops-role-original.json` to the same endpoint after the denial checks:

      ```bash
      jq '.permissions[].access |= map(select(.resource != "repository" or .action != "push"))' \
        devops-role-original.json > devops-role-no-push.json
      ```

   4. As the system administrator, begin editing a custom role and cancel the dialog. Verify that its description and permissions remain unchanged. Update the unassigned role from category 2, step 4 through `PUT /api/v2.0/roles/{role_id}` and verify that the UI reflects the saved description and permissions. Submit an invalid permission update and verify that the previously saved role remains intact.

      Save this body as `role-update.json` and submit it to `PUT /api/v2.0/roles/<role-id>`. It changes the description and adds permission to read artifact additions. Send the complete intended permissions array, including permissions that must remain granted:

      ```json
      {
        "name": "API Test Reader",
        "description": "Updated reader with access to artifact additions",
        "permissions": [
          {
            "kind": "project-role",
            "namespace": "*",
            "access": [
              {"resource": "repository", "action": "list", "effect": "allow"},
              {"resource": "repository", "action": "read", "effect": "allow"},
              {"resource": "repository", "action": "pull", "effect": "allow"},
              {"resource": "artifact", "action": "list", "effect": "allow"},
              {"resource": "artifact", "action": "read", "effect": "allow"},
              {"resource": "tag", "action": "list", "effect": "allow"},
              {"resource": "artifact-addition", "action": "read", "effect": "allow"}
            ]
          }
        ]
      }
      ```

      For the invalid update, generate and submit `role-invalid-update.json` to the same endpoint:

      ```bash
      jq '.permissions[0].access[0].resource = "unknown-resource"' \
        role-update.json > role-invalid-update.json
      ```

   5. As `project-admin`, change `devops-user` in project A from `DevOps Engineer` to `Project Reader`. Verify that the active user's next robot creation request in A is denied, pull remains allowed, and the DevOps permissions in project B remain effective. Restore the custom role in A, then remove the user's membership in A; verify that new project API requests and pulls using fresh registry authorization are denied in A while B remains accessible. Re-add the user to A for the deletion checks.

      Use `PUT /api/v2.0/projects/custom-role-a/members/<member-id>` with `{"role_id":101}`, replacing `101` first with `Project Reader`'s ID and then with `DevOps Engineer`'s ID. Remove the membership with `DELETE /api/v2.0/projects/custom-role-a/members/<member-id>` and no body. Re-add it with `POST /api/v2.0/projects/custom-role-a/members` and `{"role_id":101,"member_user":{"username":"devops-user"}}`, using the `DevOps Engineer` ID. For the robot denial, POST the robot body from category 4, step 5 with a fresh name in project A.

7. **Role deletion, persistence, and audit logging**

   1. As the system administrator, try to delete `DevOps Engineer` while it is assigned in projects A and B. Verify that the UI reports that the role is in use and that the API returns HTTP 412. Verify that the role, its permissions, and all member assignments remain intact. Remove its assignments from A only and repeat; deletion must still be blocked by the assignments in B.

      Use the recorded `DevOps Engineer` ID in this request. DELETE has no body; omit `--data` and `--data-binary`:

      ```bash
      curl -i --user admin --request DELETE \
        'https://<harbor-host>/api/v2.0/roles/<devops-role-id>'
      ```

      Remove each assignment in A with `DELETE /api/v2.0/projects/custom-role-a/members/<member-id>` and no body, then repeat the role DELETE.

   2. Before removing the remaining assignments, restart Harbor. Log in again and verify that custom role definitions, permissions, descriptions, and remaining memberships persist. Repeat an allowed action and a denied action for `devops-user` in project B, and verify that the previously created robot retains its own explicit permissions.
   3. Reassign all remaining users of `DevOps Engineer` to `Project Reader`, or remove their memberships. Delete `DevOps Engineer` as the system administrator. Verify that the deletion succeeds, a subsequent role GET returns HTTP 404, and the role is absent from the Roles list and refreshed member selectors in every project. Attempt to assign its deleted ID through the member API; verify rejection. Verify that other roles and the existing robot's explicit permissions are unaffected.

      Reassign each remaining membership with `PUT /api/v2.0/projects/<project-name>/members/<member-id>` and `{"role_id":101}`, using the actual `Project Reader` ID, or DELETE that membership with no body. Delete the role with `DELETE /api/v2.0/roles/<devops-role-id>` and no body. For the deleted-role check, POST to `/api/v2.0/projects/custom-role-a/members` with `{"role_id":101,"member_user":{"username":"api-user"}}`, replacing `101` with the recorded deleted role ID.

   4. Delete the unassigned role created through the API in category 2, step 4 and verify success. Attempt to retrieve, update, and delete its now-missing ID; verify HTTP 404 responses without server errors or changes to other roles.

      Use `DELETE /api/v2.0/roles/<role-id>` with no body before and after deletion. For the update of the missing role, use `PUT /api/v2.0/roles/<role-id>` with `role-update.json` from category 6, step 4.

   5. Go to Logs as the system administrator. Verify that successful custom role creation, modification, and deletion, and project member role changes, have audit records identifying the actor, operation, affected role or membership, and time. Compare them with the actions performed above. Rejected operations must not appear as successful changes.

8. **LDAP and OIDC group assignments**

   1. On a separate Harbor instance configured for LDAP authentication and group lookup, recreate projects A and B and the `Project Reader`, `DevOps Engineer`, and `Security Auditor` roles from category 2, step 3. Prepare a group `custom-role-team` containing two ordinary users, a second group `custom-role-readers` containing one of those users, and an unrelated user in neither group. Give these users no direct project membership and no system administrator privileges.
   2. As a project administrator, add `custom-role-team` to project A with `DevOps Engineer` and to project B with `Security Auditor`. Verify that custom roles are selectable when adding groups and when changing an existing group member's role. Log in as each group user and verify push/pull and permitted robot operations in A, scan and read access in B, and denial of push in B. Verify that the unrelated user has no access to either private project.

      For `POST /api/v2.0/projects/custom-role-a/members`, replace `101` with the `DevOps Engineer` ID and the LDAP DN with the actual test group DN. Repeat for project B using the `Security Auditor` ID:

      ```json
      {
        "role_id": 101,
        "member_group": {
          "group_name": "custom-role-team",
          "group_type": 1,
          "ldap_group_dn": "cn=custom-role-team,ou=groups,dc=example,dc=com"
        }
      }
      ```

   3. As a project administrator, add `custom-role-readers` to project A with `Project Reader`. Verify that the overlapping user retains the permissions granted by both memberships. Keep both users logged in and, as the system administrator, remove robot creation permission from the shared `DevOps Engineer` definition; verify that their next robot creation requests in A are denied. Restore the permission and verify that new requests succeed without another login.

      POST to `/api/v2.0/projects/custom-role-a/members` using step 2's group body with `group_name: "custom-role-readers"`, that group's actual `ldap_group_dn`, and the `Project Reader` ID. For the role revocation and restoration PUT requests, use the body generation in category 6, step 1 and restore the saved original role as in category 6, step 2. Use category 4, step 5's robot POST body with a fresh name for each group user's permission check.

   4. As a project administrator, change `custom-role-team` in A to `Security Auditor`. Verify that its users gain scan permission and lose robot creation and push permissions in A; use new registry authorization for Docker checks. Remove this group's project A membership. Verify that the overlapping user retains only `Project Reader` access through the second group, while the other user loses access to A. Their group assignment in B must remain effective.

      Send `PUT /api/v2.0/projects/custom-role-a/members/<group-member-id>` with `{"role_id":101}`, replacing `101` with the `Security Auditor` ID. Use the group's project membership ID. Then send `DELETE /api/v2.0/projects/custom-role-a/members/<group-member-id>` with no body.

   5. As the system administrator, attempt to delete `Security Auditor` while it is still assigned to the group in B. Verify that deletion is rejected. Remove that group assignment and any other assignments, then verify that deletion succeeds. Repeat the group tests in this category on an OIDC deployment with group claims configured to verify the supported OIDC group assignment path.

      Use `DELETE /api/v2.0/roles/<security-auditor-role-id>` with no body before and after removing group assignments. Remove a group assignment with `DELETE /api/v2.0/projects/<project-name>/members/<group-member-id>` and no body.

      When repeating the group tests on OIDC, use this body for `POST /api/v2.0/projects/custom-role-a/members`. Set `role_id` to the relevant custom role and use the configured group claim value. Group updates and removals use the same PUT and DELETE requests as step 4:

      ```json
      {
        "role_id": 101,
        "member_group": {
          "group_name": "custom-role-team",
          "group_type": 3
        }
      }
      ```

9. **Upgrade compatibility and cleanup**

   1. On a separate upgrade test environment, install a supported predecessor release and record user/group assignments for all five built-in roles, representative allowed and denied actions, and existing robot permissions. Upgrade through the supported procedure to the feature build. Verify that the assignments, built-in role IDs and permissions, robot permissions, and existing member API behavior are preserved. Verify that built-in roles are protected and that a newly created custom role can be assigned and enforced after the upgrade.
   2. Remove the disposable users, group memberships, robot accounts, images, projects, and unassigned custom roles created by this testcase. Verify that cleanup succeeds without modifying built-in roles or unrelated resources. Record any failures with the Harbor build, actor, project, role ID, selected permissions, request/response, and expected result.

      For API cleanup, use these DELETE endpoints without request bodies. Remove artifacts and repositories before deleting their project, and remove memberships before deleting their custom role.

      | Resource | Endpoint for `DELETE` |
      | --- | --- |
      | User or group membership | `/api/v2.0/projects/<project-name>/members/<member-id>` |
      | Robot | `/api/v2.0/robots/<robot-id>` |
      | Artifact | `/api/v2.0/projects/<project-name>/repositories/<repository>/artifacts/<reference>` |
      | Repository | `/api/v2.0/projects/<project-name>/repositories/<repository>` |
      | Immutability rule | `/api/v2.0/projects/<project-name>/immutabletagrules/<rule-id>` |
      | Disposable user | `/api/v2.0/users/<user-id>` |
      | Empty test project | `/api/v2.0/projects/<project-name>` |
      | Unassigned custom role | `/api/v2.0/roles/<role-id>` |
