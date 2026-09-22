# Proposal: Architecture/Platform Filter for Replication

- **Issue**: [goharbor/harbor#19864](https://github.com/goharbor/harbor/issues/19864) — "How to replicate specified arch images from DockerHub?"
- **Status**: Draft for discussion
- **Author**: generated with Claude Code, based on repo state at `main`

## 1. Problem statement

Multi-arch images (Docker manifest list / OCI image index) bundle several
platform-specific manifests (`linux/amd64`, `linux/arm64/v8`, `linux/arm/v7`,
`windows/amd64`, …) under one tag. Today, a Harbor replication rule can only
filter on **resource type / repository name / tag / label** — there is no way
to say "replicate this repository, but only the `linux/amd64` and
`linux/arm64` variants." As a result, users who only need 1-2 platforms end up
replicating (and storing) every platform Docker Hub publishes for an image,
which wastes bandwidth and registry storage, especially over slow links or in
air-gapped environments where bandwidth is the whole point of using
replication.

This has been a live, +1'd request since 2023, and a maintainer
(`stonezdj`) already identified the core technical obstacle in the issue
thread:

> Harbor registry will validate the manifest's integrity at push, all
> sub-images and blobs described in the manifest should exist in the registry
> before the manifest list (parent) push. The problem is when partial of the
> manifest list's sub-images are replicated to harbor, in order to pass the
> integrity check, the final manifest content of the manifest list should be
> trimmed before push, like we did in the proxy cache. It will cause the
> replication job always rerun when source/target artifact digest mismatch.
>
> Suggest to use proxy cache instead of replication, because proxy cache
> always caches the arch it has been pulled.

This document proposes a concrete design that solves that obstacle, so
replication (not just proxy cache) can support platform filtering.

## 2. Current architecture (relevant pieces)

| Concern | Location |
|---|---|
| Filter types (`resource`/`name`/`tag`/`label`) | `src/pkg/reg/model/policy.go:21-88` (`Filter{Type, Value, Decoration}`) |
| Filter application against fetched artifacts | `src/pkg/reg/filter/artifact.go` (`BuildArtifactFilters`, `artifactTagFilter`, `artifactLabelFilter`) |
| Repository-level filtering | `src/pkg/reg/filter/repository.go` |
| Resource re-assembly after filtering | `src/pkg/reg/filter/resource.go` (`DoFilterResources`) |
| Candidate-artifact model | `src/pkg/reg/model/resource.go:29-66` (`Resource`, `ResourceMetadata`, `Artifact{Type, Digest, Labels, Tags, IsAcc, ParentTags}`) |
| Adapter capability advertisement (drives the rule-editor UI) | `RegistryInfo.SupportedResourceFilters` — e.g. `src/pkg/reg/adapter/harbor/base/adapter.go:101-134` |
| Actual blob/manifest transfer | `src/controller/replication/transfer/image/transfer.go` — `copyArtifact()` (pulls manifest, then **unconditionally** walks `manifest.References()`, `:210-257`) and `copyContent()` (`:260-285`, recurses into nested indexes / copies blobs) |
| Local manifest-list trimming precedent | `src/controller/proxy/manifestcache.go` — `ManifestListCache.updateManifestList()` (`:113-129`) and `.push()` (`:131-172`) |
| Per-platform data already parsed & stored locally | `src/pkg/artifact/model.go:139-186` (`Reference{ChildDigest, Platform *v1.Platform}`), populated by `src/controller/artifact/manifest/index.go:69-84` when Harbor abstracts an index |
| Per-platform data for **non-list** (single-manifest) images | `src/controller/artifact/processor/image/manifest_v2.go:65-66` and `manifest_v1.go:49` — `artifact.ExtraAttrs["architecture"]` / `["os"]`, parsed straight from the image config, no index involved |

**The gap**: `model.Artifact` (the unit replication filters operate on) has no
platform/architecture field at all, and the transfer engine copies whatever
`manifest.References()` returns with no pruning. Platform data already exists
in Harbor's own DB (`pkg/artifact.Reference.Platform`) but is dropped when the
Harbor→Harbor adapter builds `model.Artifact` (`src/pkg/reg/adapter/harbor/v2/client.go:88-100`
recurses into `artItem.References` only to find accessories, discarding
`ref.Platform`). For third-party (generic Docker v2) sources, the `native`
adapter doesn't even pull manifests during listing
(`src/pkg/reg/adapter/native/adapter.go:216-226` builds `model.Artifact` from
tag names alone), so platform data isn't available pre-filter for those
sources today either.

The proxy cache already solved the "push a manifest list that only lists
what's actually present" problem for its own use case
(`ManifestListCache.updateManifestList`, using
`manifestlist.FromDescriptors(existMans)` to rebuild the list and get a new,
self-consistent digest, then reconciling tag/digest lookups against the new
digest in `EnsureTag()`, `src/controller/proxy/controller.go:100-124`). That
mechanism is the template this proposal reuses for replication.

## 3. Two kinds of "platform filtering" — worth separating

There are actually two distinct cases, with very different implementation
cost, and the design should treat them differently:

1. **Single-platform (non-index) artifacts.** Docker Hub's `library/php`
   also has environments where a given tag simply *is* one platform
   manifest, and Harbor already parses `ExtraAttrs["architecture"]` /
   `["os"]` for these from the image config at abstraction time. Filtering
   these is a simple **include/exclude the whole artifact** decision — no
   manifest rewriting, no digest recomputation, same shape as the existing
   tag/label filters.
2. **Manifest-list (index) artifacts.** These are the case the issue is
   actually about: one tag, N platform children. Filtering "amd64 +
   arm64/v8 only" out of `[amd64, arm/v6, arm/v7, arm64/v8, 386, ppc64le,
   s390x]` means the *artifact as a whole should still replicate*, but the
   manifest list pushed to the destination must be **rebuilt to only
   reference the platforms that were actually copied**, exactly like the
   proxy cache does. This is the part with digest-recomputation and
   integrity-check implications.

The design below handles both, but flags case 2 as the higher-effort,
higher-value part.

## 4. Proposed design

### 4.1 New filter type: `platform`

Add a fourth-ish filter type (mirroring the existing `resource`/`name`/`tag`/`label`
shape) so it plugs into the existing filter list UI/API with minimal net-new
plumbing:

```go
// src/pkg/reg/model/policy.go
const (
    FilterTypeResource = "resource"
    FilterTypeName     = "name"
    FilterTypeTag      = "tag"
    FilterTypeLabel    = "label"
    FilterTypePlatform = "platform" // NEW
)
```

`Filter.Value` for this type is a `[]string` of `os/arch[/variant]` strings,
e.g. `["linux/amd64", "linux/arm64/v8"]` — the same triple Docker/OCI already
uses in `v1.Platform{OS, Architecture, Variant}`, so no new vocabulary is
invented. Validation in `Filter.Validate()` (`policy.go:43-88`) gets a new
case requiring a non-empty `[]string` of well-formed `os/arch[/variant]`
entries.

### 4.2 Extend the candidate-artifact model

```go
// src/pkg/reg/model/resource.go
type Artifact struct {
    Type       string
    Digest     string
    Labels     []string
    Tags       []string
    IsAcc      bool
    ParentTags []string

    // NEW
    Platform   *v1.Platform          // set for single-manifest artifacts
    References []ArtifactReference   // set for manifest-list/index artifacts
}

type ArtifactReference struct {
    ChildDigest string
    Platform    *v1.Platform
}
```

`v1.Platform` is the existing `github.com/opencontainers/image-spec/specs-go/v1`
type already used by `pkg/artifact.Reference.Platform` — reusing it avoids a
parallel type.

### 4.3 Populate platform data when listing artifacts

- **Harbor→Harbor adapter** (`src/pkg/reg/adapter/harbor/v2/client.go:88-100`):
  when walking `artItem.References`, copy `ref.Platform` (and `ref.ChildDigest`)
  onto the new `ArtifactReference` list instead of discarding it. For a
  single-manifest artifact, copy `artItem.ExtraAttrs["architecture"]` /
  `["os"]` into `Artifact.Platform`. This is a pure "stop dropping data we
  already have" change — no new registry calls, no new cost.
- **Generic Docker v2 / `native` adapter** (`src/pkg/reg/adapter/native/adapter.go:216-226`):
  currently builds `model.Artifact` from tag names only, without pulling any
  manifest. To support platform filtering against third-party registries
  (Docker Hub etc., the actual case in the issue), this adapter needs to
  additionally `HEAD`/`GET` the manifest for each tag **only when the policy
  being evaluated has a `platform` filter configured** (checked once, passed
  down as a flag/option into `FetchArtifacts`), so repositories without a
  platform filter pay zero extra cost. When the fetched manifest is an
  index/manifest-list, parse its `manifests[].platform` entries directly
  (no need to pull each child manifest — the list descriptor already carries
  `platform`) into `Artifact.References`. When it's a single manifest, an
  extra blob GET for the image config would be needed to get
  `architecture`/`os` cheaply skip this for v1 and treat single-manifest
  artifacts as "unknown platform → always included" if the extra round trip
  is judged too costly; this is a tunable tradeoff to flag explicitly in
  review.
- Other adapters (`dtr`, `jfrog`, `aliacr`, `dockerhub`, `githubcr`,
  `tencentcr`) can follow the same opt-in pattern as they're prioritized;
  Phase 1 (§4.7) only requires the Harbor and generic-v2/DockerHub adapters.

### 4.4 Filtering logic (`src/pkg/reg/filter/artifact.go`)

Add `artifactPlatformFilter` and wire it into `BuildArtifactFilters`'s switch
(the existing unregistered `artifactTypeFilter`/`artifactTaggedFilter` in
this file already show the "filter struct exists, just add a `case`" pattern
to follow). Unlike the existing filters (all-or-nothing on the whole
`Artifact`), this one has to branch on artifact shape:

```go
func (f *artifactPlatformFilter) Filter(candidates []*model.Artifact) ([]*model.Artifact, error) {
    var result []*model.Artifact
    for _, art := range candidates {
        switch {
        case len(art.References) > 0: // manifest list / index
            matched := matchReferences(art.References, f.platforms)
            if len(matched) == 0 {
                continue // no requested platform present -> drop the whole tag
            }
            cp := art.Copy()
            cp.References = matched
            cp.NeedsTrim = len(matched) < len(art.References) // NEW flag, see 4.5
            result = append(result, cp)
        case art.Platform != nil: // single-manifest artifact
            if platformMatches(art.Platform, f.platforms) {
                result = append(result, art)
            }
        default: // platform unknown (e.g. adapter didn't resolve it) -> keep, don't silently drop
            result = append(result, art)
        }
    }
    return result, nil
}
```

This is applied per-adapter inside `FetchArtifacts` exactly like the existing
tag/label filters (`harbor/v2/adapter.go:120,130`, `native/adapter.go:213,227`,
etc.), and also flows through the event-triggered path via
`filter.DoFilterResources` (`src/pkg/reg/filter/resource.go`, called from
`src/controller/event/handler/replication/event/handler.go:92`) since both
paths share the same filter package.

### 4.5 Transfer-time manifest-list trimming

This is the piece that makes the filter actually save storage/bandwidth
instead of just hiding platforms in the UI. In
`src/controller/replication/transfer/image/transfer.go`:

- `copyArtifact()` (`:210-257`) currently pulls the manifest (`:214`) and
  unconditionally loops over `manifest.References()` (`:243-247`) to copy
  every child. Change:
  1. After pulling the manifest, check whether the `model.Artifact` being
     copied carries a `NeedsTrim`/`References` subset from the platform
     filter (threaded down from §4.4 via the job's resource payload).
  2. If so, and the manifest type-switches to
     `*manifestlist.DeserializedManifestList` (or the OCI
     `*ocischema.DeserializedImageIndex` equivalent), rebuild it using only
     the matched child descriptors — **identical approach to**
     `src/controller/proxy/manifestcache.go:updateManifestList()`, i.e.
     `manifestlist.FromDescriptors(matchedDescriptors)` (or the OCI-index
     analogue) to get a new manifest object and a **new digest**.
  3. Only copy the blobs/child manifests for the retained platforms (loop
     over the trimmed reference list, not `manifest.References()`) — this is
     where the actual bandwidth/storage savings come from.
  4. Push the *trimmed* manifest list to the destination instead of the
     original payload.
- Because the pushed digest now differs from the source digest **by
  design**, the tag created at the destination must point at the new
  (trimmed) digest, not the source one — mirror the digest-substitution
  approach in `src/controller/proxy/controller.go:EnsureTag()` (lookup
  original→trimmed digest before tagging), but simpler here since
  replication doesn't need a long-lived Redis-backed mapping — it can just
  use the digest it computed in the same request/job.

**Edge cases to make explicit in implementation:**
- *All platforms filtered out*: skip the artifact/tag entirely (don't push
  an empty manifest list); surface this in the replication execution log so
  it's visible ("skipped `php:8.3` — no platform in the source manifest list
  matched the configured filter") rather than silently vanishing.
- *Only one platform remains after trimming*: keep pushing it **as a
  manifest list with one entry**, not unwrapped into a bare single-platform
  manifest. This preserves the tag's content-type/shape for any client that
  already pulled it as an index, and avoids a second special case in the
  push path.
- *Non-Harbor destinations*: the trimming logic operates purely on the
  standard `docker/distribution` manifest-list types already in use, so it
  is destination-adapter-agnostic — no changes needed on the push side for
  non-Harbor registries.

### 4.6 Idempotency / re-run behavior

This is the specific risk `stonezdj` called out: normal replication decides
"already replicated, skip" by comparing source and destination digests
directly. Once trimming makes the destination digest intentionally diverge
from the source digest, that comparison breaks and would cause the rule to
re-copy on every run.

Fix: when trimming occurs, record the mapping (source manifest-list digest →
trimmed digest actually pushed) as part of the replication task/execution
record (a new column or JSON field on the existing task row, scoped to that
execution — no need for a cross-execution cache like proxy cache's Redis
entry, since replication re-derives this each run from the current source
state anyway). The "should I skip this artifact" pre-check then compares
source-digest-if-untrimmed OR looks up the last-known trimmed-mapping for
that source digest, instead of doing a raw digest equality check.

### 4.7 API / swagger changes

- `api/v2.0/swagger.yaml`: `ReplicationFilter.type` (currently a free-form
  string, `:7738-7749`) documentation gets `platform` added to the described
  values; no schema-breaking change since `type`/`value` are already
  loosely typed.
- `RegistryInfo.supported_resource_filters` (`:7885-7889`, `:7948-7963`):
  the Harbor adapter's `Info()` (`src/pkg/reg/adapter/harbor/base/adapter.go:101-134`)
  declares a new `FilterStyle{Type: "platform", Style: FilterStyleTypeList,
  Values: [...]}` — Phase 1 ships a static well-known list (`linux/amd64`,
  `linux/arm64`, `linux/arm/v7`, `linux/arm/v6`, `linux/386`, `linux/ppc64le`,
  `linux/s390x`, `windows/amd64`) plus free-text entry for anything else,
  same UX precedent as the existing `label` filter's checkbox-list-plus-autocomplete
  pattern.

### 4.8 Frontend (Angular)

- `src/portal/src/app/shared/entities/shared.const.ts:104-109`: add
  `PLATFORM: 'platform'` to the `FilterType` const enum.
- `create-edit-rule.component.ts`: `initFilter()` (`:463-481`) gets a branch
  for `platform` mirroring the `label` branch (array-valued `FormArray`), and
  `setFilterAndTrigger()` (`:668-672`) needs no change since
  `supportedFilters` already comes straight from the adapter's
  `/registries/{id}/info` response.
- `create-edit-rule.component.html`: reuse the existing checkbox-list pattern
  used for `label` (`:247-335`) — a checkbox per well-known platform string
  plus a free-text "add custom platform" input for anything not in the
  static list (e.g. less common variants).
- i18n: add `REPLICATION.FILTER_PLATFORM` (label) and a tooltip string
  explaining the manifest-list-trimming behavior described in §4.5, so users
  understand the destination digest will differ from the source for
  filtered multi-arch tags.

### 4.9 Phased rollout

1. **Phase 1 — Harbor → Harbor.** Platform data is already in Harbor's DB
   (`pkg/artifact.Reference.Platform`), so this phase is "stop dropping data
   we have" (§4.3) + filter (§4.4) + trim-on-push (§4.5) + idempotency
   fix (§4.6). No new registry round trips, no adapter-specific rate-limit
   concerns. This alone covers the common "sync between two Harbor instances,
   only keep amd64+arm64" use case.
2. **Phase 2 — Generic Docker v2 sources (Docker Hub, GHCR, etc.), the case
   in the issue.** Add the opt-in manifest-list fetch in the `native`
   adapter (§4.3) gated on the policy actually having a `platform` filter.
   Explicitly document the added per-tag API call and its interaction with
   source-registry rate limits (Docker Hub in particular) in the release
   notes / rule-editor tooltip.
3. **Phase 3 — Remaining adapters** (`dtr`, `jfrog`, `aliacr`, `dockerhub`
   dedicated adapter, `githubcr`, `tencentcr`) get the same treatment as
   demand warrants, following the Phase 2 pattern per adapter.

## 5. Alternatives considered

| Option | Pros | Cons |
|---|---|---|
| **A. This proposal** (filter + transfer-time trim, phased) | Solves the actual ask: scheduled/triggered bulk replication with real storage savings; reuses proven trimming logic from proxy cache | Non-trivial: touches filter model, adapters, transfer engine, and replication idempotency bookkeeping |
| **B. Point users at Proxy Cache** (maintainer's stated workaround, no code change) | Zero engineering cost; proxy cache already discards un-pulled platforms | Passive/pull-driven only — doesn't fit scheduled or push-triggered sync, air-gapped, or "pre-warm before traffic arrives" use cases, which is what replication is for; doesn't help users who explicitly asked for a replication-time filter |
| **C. Allow Harbor to accept "sparse" manifest lists directly at push** (relax the integrity check core-wide, let replication push the untrimmed source list and rely on lazy/partial storage) | Avoids digest recomputation entirely | Much larger blast radius — changes core registry-integrity semantics for *all* pushes, not just replication, with OCI/Docker spec-compliance risk; higher review/security bar for a narrower payoff |
| **D. Replicate everything, GC unwanted platforms afterward** | Simple, no filter/transfer changes | Defeats the purpose (still pays full bandwidth cost); still needs the trim-and-redigest logic to make destination manifest lists self-consistent after GC, so it doesn't actually avoid the hard part |

Recommendation: **Option A**, phased as in §4.9, with Option B documented in
the meantime (and in docs going forward) as the lower-effort workaround for
users who can tolerate pull-driven caching instead of push-driven
replication.

## 6. Open questions for maintainers

1. Should the trimmed-digest mapping (§4.6) live on the replication task
   row, or would a lighter approach (re-derive by re-fetching the source
   manifest list each run and comparing platform sets, skipping the mapping
   store entirely) be preferred to avoid a schema change?
2. For Phase 2, is an extra per-tag manifest-list GET against the source
   registry (only when a platform filter is configured) acceptable, or
   should this be batched/cached more aggressively given Docker Hub's rate
   limits?
3. Should `platform` filtering compose with the existing `Decoration`
   (matches/excludes) concept the same way `tag`/`label` do, or is
   include-only (a positive list of platforms to keep) sufficient for v1?
4. Single-manifest artifacts with **no** resolvable platform info (adapter
   didn't fetch it, e.g. Phase 2 not yet implemented for that adapter):
   this proposal defaults to "keep it" (fail open) rather than silently
   dropping artifacts a user might not expect to lose — confirm this is the
   desired default.
