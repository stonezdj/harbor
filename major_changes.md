# Major Changes in Harbor `release-2.16.0` (compared to `release-2.15.0`)

## Overview

This document summarizes the major changes introduced in Harbor branch `release-2.16.0` that are **unique to `release-2.16.0`** (not backported to `release-2.15.0`).

### Selection Criteria
- **Included:**
  - Commits modifying `make/harbor.yml.tmpl` / `harbor.yaml.tmpl`
  - Commits introducing new configuration options, system settings, or schema changes
  - Commits impacting the Harbor UI / Portal
- **Excluded:**
  - Dependency and version bump commits (e.g., updates limited to `go.mod`, `go.sum`, `package.json`, `package-lock.json`, Dockerfiles, or CI workflows)
  - Pure translation / i18n message updates (`src/portal/src/i18n/lang/*.json`)
  - Commits cherry-picked or backported into `release-2.15.0`

---

## 1. Updates to `harbor.yaml.tmpl` (`make/harbor.yml.tmpl`)

### 1. Webhook Egress Policy Levels & Allowlist
- **Commit:** `f6f5bac8d1` | [PR #24079](https://github.com/goharbor/harbor/pull/24079)
- **Summary:** Replaces single SSRF boolean with granular egress policy levels (`allow_all`, `block_restricted`, `public_only`) and an explicit target allowlist.
- **Harbor YAML Changes:**
  ```yaml
  webhook_egress:
    policy: block_restricted # allow_all | block_restricted | public_only
    allowlist:
      - 10.20.0.0/16
      - "*.svc.cluster.local"
  ```
- **Configuration Items:** `HARBOR_WEBHOOK_EGRESS_POLICY`, `HARBOR_WEBHOOK_EGRESS_ALLOWLIST`, `webhook_egress.policy`, `webhook_egress.allowlist`.

### 2. Default Private Network Access for Webhooks
- **Commit:** `9a37bcf55f` | [PR #24078](https://github.com/goharbor/harbor/pull/24078)
- **Summary:** Defaults `HARBOR_ALLOW_PRIVATE_NETWORK_ACCESS` to `true` to prevent breaking existing on-premise and Kubernetes deployments where webhooks target internal cluster endpoints.

### 3. Webhook SSRF Guard & Private Network Isolation
- **Commit:** `b323c0c8ec`
- **Summary:** Introduced dial-time private-network SSRF protection and `network.allow_private_network_access` setting in template and migration scripts.

### 4. Registry Component Proxy Support
- **Commit:** `f89aad3720` | [PR #23437](https://github.com/goharbor/harbor/pull/23437)
- **Summary:** Adds `registry` to `proxy.components` in `make/harbor.yml.tmpl` and Helm chart values so that the registry container receives proxy environment variables (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`).

### 5. Metric Enablement Migration Formatting
- **Commit:** `da420bcad0` | [PR #22580](https://github.com/goharbor/harbor/pull/22580)
- **Summary:** Normalizes `metric.enabled` values to lowercase across migration templates during upgrades from older Harbor releases.

---

## 2. Introduced New Configuration Items & Settings

### 1. Custom Project Roles & Hybrid RBAC
- **Commit:** `37dc02fdff` | [PR #23804](https://github.com/goharbor/harbor/pull/23804)
- **Summary:** Introduces database migration `0190_2.16.0_schema.up.sql` and RBAC permission models allowing creation and configuration of custom project roles alongside built-in roles.
- **Configuration / API:** New REST API endpoints under `/roles` and project member role assignments.

### 2. Proxy Cache Repository Filter
- **Commit:** `266aad1eca` | [PR #23527](https://github.com/goharbor/harbor/pull/23527)
- **Summary:** Implements repository filtering API (`proxy_cache_repo_filter`) supporting pattern matching (wildcard and regex) to selectively allow or restrict caching of repositories.

### 3. Advanced Proxy Cache Configurations (Connection Limit & TTL)
- **Commits:** `907173f70f` ([PR #23536](https://github.com/goharbor/harbor/pull/23536)), `89dbf8fbf7` ([PR #22678](https://github.com/goharbor/harbor/pull/22678))
- **Summary:** Adds `max_upstream_conn` (upstream registry max concurrent connection limit) and cache TTL configurations to proxy-cache project settings.

### 4. Enable Proxy Cache Referrer API
- **Commit:** `63618e8a9f` | [PR #23151](https://github.com/goharbor/harbor/pull/23151)
- **Summary:** Adds project metadata option `enable_proxy_cache_referrer_api` to proxy OCI 1.1 Referrer API requests upstream with paging and local caching.

### 5. Retention Setting During Proxy-Cache Project Creation
- **Commit:** `de749f4e50` | [PR #23863](https://github.com/goharbor/harbor/pull/23863)
- **Summary:** Exposes tag retention policy configuration directly in the project creation dialog for proxy-cache projects.

### 6. Cache Backend Transition (Redis to Valkey)
- **Commit:** `88ab66244e` | [PR #23157](https://github.com/goharbor/harbor/pull/23157)
- **Summary:** Replaces Redis base image and service container with Valkey across Docker Compose, photon prepare templates, and service configs.

### 7. Non-Empty `robot_name_prefix` Constraint
- **Commit:** `68fffc61b1` | [PR #22556](https://github.com/goharbor/harbor/pull/22556)
- **Summary:** Enforces non-empty `robot_name_prefix` in system configuration metadata to prevent OIDC CLI logins from being blocked.

---

## 3. Major UI Framework Upgrades & UI Features

### 1. Major UI Framework Upgrade (Angular v21 & Clarity v18)
- **Commit:** `a821bae24a` | [PR #23078](https://github.com/goharbor/harbor/pull/23078)
- **Summary:** Major frontend framework upgrade across 380+ portal files upgrading Angular to v21, Clarity Design System to v18, and Node.js to v22.

### 2. Custom Project Roles Frontend UI
- **Commit:** `205159bd91` | [PR #23970](https://github.com/goharbor/harbor/pull/23970)
- **Summary:** Adds dedicated navigation entry, custom role creation dialog, permission selection matrix, and member assignment views.

### 3. Aggregated Task Error Details & Execution Failure UI
- **Commit:** `1edae4acdb` | [PR #24028](https://github.com/goharbor/harbor/pull/24028)
- **Summary:** Surfaced aggregated task failure details, execution error messages, and log viewers directly in the replication and job execution UI.

---

## 4. UI Enhancements, Form Updates & Bug Fixes

| Commit | PR | Component | Description |
| :--- | :--- | :--- | :--- |
| `c3ec2b133e` | [PR #23540](https://github.com/goharbor/harbor/pull/23540) | Artifact Table | Adds signature inheritance badge indicator for child artifacts under multi-arch OCI index rows. |
| `23bfa9a110` | [PR #24056](https://github.com/goharbor/harbor/pull/24056) | P2P Preheat | Preserves multi-value filter inputs when editing existing P2P preheat policies. |
| `c577196d88` | [PR #22708](https://github.com/goharbor/harbor/pull/22708) | Project Detail | Dynamically displays or hides project navigation tabs according to current user's permissions. |
| `4e30db31aa` | [PR #23393](https://github.com/goharbor/harbor/pull/23393) | Portal Components | Fixes layout alignment, CVE export dialog, distribution setup, and scanner configurations across 26 UI files. |
| `1cd644ca39` | [PR #23607](https://github.com/goharbor/harbor/pull/23607) | Artifact List | Resolves tag column text truncation and count badge overlapping. |
| `70288fa699` | [PR #23646](https://github.com/goharbor/harbor/pull/23646) | Artifact Grid | Adds column width resizing support for the SBOM column in artifact tables. |
| `1a83ce4ece` | [PR #22681](https://github.com/goharbor/harbor/pull/22681) | Project Policy | Fixes `max_upstream_conn` input validation and disabled control state bindings. |
| `1a236ce8cb` | [PR #22980](https://github.com/goharbor/harbor/pull/22980) | Session & Polling | Configures background polling requests (GC, audit log purge, task status) to not falsely renew user session TTL. |
| `a26dea01da` | [PR #22694](https://github.com/goharbor/harbor/pull/22694) | Project Policy | Fixes two-way data bindings and form validation in project policy settings. |
| `bd4c15d2aa` | [PR #22042](https://github.com/goharbor/harbor/pull/22042) | Dashboard | Aligns CSS grid and summary cards in the project statistics dashboard panel. |
| `6084e31d16` | [PR #23859](https://github.com/goharbor/harbor/pull/23859) | Swagger UI | Prevents Webpack HTML minification from breaking the Swagger UI loading page. |
| `95b28b3cd4` | [PR #23889](https://github.com/goharbor/harbor/pull/23889) | Endpoint Forms | Handles URL protocol schemes and endpoints case-insensitively in distribution and scanner modal forms. |
