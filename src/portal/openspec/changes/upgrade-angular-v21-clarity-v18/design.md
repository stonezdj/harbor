## Context

The portal is an Angular 19 application using NgModule-based architecture with 35+ feature modules. It uses Clarity 17 for UI components and icons. The build pipeline relies on the legacy `@angular-devkit/build-angular:browser` builder (deprecated in Angular 17, superseded by the `application` builder backed by Vite + esbuild). Icon assets from `@clr/icons` are loaded as global scripts and stylesheets in `angular.json`.

**Current key versions:**
- Angular: `^19.2.x`, Angular CLI / devkit: `^19.2.x`
- `@clr/angular` / `@clr/ui`: `17.12.4`
- `@clr/icons`: `13.0.2` (deprecated upstream; removed in Clarity 18)
- TypeScript: `~5.8.3`, `moduleResolution: "node"` (deprecated since TS 4.7)
- Zone.js: `^0.15.x`

**Scope constraints:**
- The project has 35+ NgModules. No standalone component migration is in scope for this upgrade.
- Tests run via Karma (unit) and Cypress (E2E); both build pipelines must remain green.

## Goals / Non-Goals

**Goals:**
- Upgrade all `@angular/*` packages to `^21.0.0` and Angular tooling to match
- Upgrade `@angular-eslint/*` to `^21.0.0`
- Upgrade `@clr/angular` and `@clr/ui` to `18.0.0`
- Migrate away from the removed `@clr/icons` package to `@cds/core` icon registration
- Migrate the build from the deprecated `browser` builder to the `application` builder
- Update `tsconfig.json` as required by Angular 21 and the new builder
- Ensure all unit and E2E tests continue to pass after the upgrade

**Non-Goals:**
- Migrating NgModules to standalone components (separate initiative)
- Adopting Signals-based APIs or zoneless mode
- Upgrading other third-party libraries (Highcharts, Prism, marked, etc.)
- Introducing new user-facing features

## Decisions

### 1. Migrate from `browser` builder to `application` builder

**Decision:** Replace `@angular-devkit/build-angular:browser` with `@angular-devkit/build-angular:application` in `angular.json` for the `build` target, and update the `serve` target to use `@angular-devkit/build-angular:dev-server`.

**Rationale:** The `browser` builder was deprecated in Angular 17 and is slated for removal in Angular 21. The `application` builder is the Angular-maintained successor; it uses Vite + esbuild for development and esbuild for production, producing faster cold-start builds and smaller bundles.

**Impact on `angular.json` options:**
| Old option | New equivalent |
|---|---|
| `aot: false` (dev) | Remove — `application` builder is always AOT |
| `main` | `browser` |
| `polyfills: "zone.js"` | `polyfills: ["zone.js"]` (array) |
| `vendorChunk`, `namedChunks`, `buildOptimizer` | Remove — not supported |
| `scripts: [...]` | Move to `externalDependencies` or load via import in `main.ts` |
| `styles: ["node_modules/@clr/icons/clr-icons.min.css"]` | Remove `@clr/icons` entries |

The lazy theme bundles (`dark-theme.scss`, `light-theme.scss`) with `inject: false` are supported in the `application` builder.

**Alternative considered:** Keep the `browser` builder using `@angular-devkit/build-angular` v21 compatibility shim if it still ships one. Rejected because the shim may have limited support lifetime and the team would need to migrate again later.

### 2. Remove `@clr/icons` and migrate to `@cds/core` icon registration

**Decision:** Remove `@clr/icons` from `package.json`. Remove its CSS and JS script references from `angular.json`. Update `src/app/shared/shared.module.ts` (the only TypeScript file importing from `@clr/icons`) to import and register icons via `@cds/core/icon`.

**Rationale:** Clarity 18 removes `@clr/icons` entirely; `@cds/core` (already a transitive dependency via `@clr/angular`) is the canonical icon system. The migration surface is small — only one `.ts` file references `@clr/icons` directly; the CSS and JS are loaded as global scripts in `angular.json`.

**Migration pattern:**
```typescript
// Before (shared.module.ts)
import '@clr/icons';
import '@clr/icons/shapes/all-shapes';

// After
import { ClarityIcons, allShapes } from '@cds/core/icon';
ClarityIcons.addIcons(...allShapes);
```

**Alternative considered:** Keep `@clr/icons` pinned at 13.x alongside Clarity 18 packages. Rejected because Clarity 18's peer-dependency resolution explicitly removes `@clr/icons`, causing install warnings and potential runtime conflicts.

### 3. Update `tsconfig.json` `moduleResolution` to `"bundler"`

**Decision:** Change `"moduleResolution": "node"` → `"moduleResolution": "bundler"` and add `"esModuleInterop": true`.

**Rationale:** Angular 17+ recommends `moduleResolution: "bundler"` when using the `application` builder (Vite/esbuild). The legacy `"node"` resolution is deprecated in TypeScript 5+ and will produce warnings. `"bundler"` aligns with how esbuild resolves modules.

**Other `tsconfig.json` changes:**
- `"module": "es2020"` → `"module": "ES2022"` (consistent with `target`)
- `"lib"` remains `["es2018", "dom"]` unless Angular 21 requires a higher baseline
- `"downlevelIteration": true` can be removed (redundant for `target: ES2022`)
- `"experimentalDecorators": true` stays until NgModule decorator syntax is migrated

**Alternative considered:** Use `"moduleResolution": "node16"`. Rejected because `"bundler"` is the Angular-recommended choice for bundler-based tools and avoids requiring explicit `.js` extension in imports.

### 4. Keep NgModule architecture; no standalone migration

**Decision:** Do not convert any NgModule to a standalone component/directive/pipe as part of this upgrade.

**Rationale:** There are 35+ NgModules in the codebase. Converting them is a large, orthogonal refactor that carries its own risk. Angular 21 fully supports NgModule-based applications. Mixing the two migrations would make rollback harder and obscure the root cause of any regressions.

### 5. Use `ng update` for automated Angular migration schematics

**Decision:** Run `ng update @angular/core@21 @angular/cli@21` to apply official schematics before making manual changes. Apply Clarity migration schematics if available for `@clr/angular@18`.

**Rationale:** Angular and Clarity publish schematic migrations that automatically rename deprecated APIs, update imports, and patch configuration files. Running them first reduces manual churn and ensures changes follow the official upgrade path.

**Order of operations:**
1. `ng update @angular/core@21 @angular/cli@21`
2. `ng update @angular/cdk@21`
3. `ng update @angular-eslint/schematics@21`
4. Manually upgrade `@clr/angular@18.0.0`, `@clr/ui@18.0.0`
5. Manually remove `@clr/icons`, add `@cds/core` if not already present
6. Migrate `angular.json` builder
7. Migrate `tsconfig.json`
8. Fix remaining compilation errors

## Risks / Trade-offs

- **AOT-only dev builds reveal hidden template errors** → Run `ng build` with AOT early to surface template type-checking errors that JIT mode silently ignores. Treat each error as a bug fix rather than upgrade friction.

- **`application` builder output path change** → The builder emits to `dist/harbor-portal/browser/` by default rather than `dist/`. Update any Docker build scripts or CI steps that copy from the `dist/` folder.

- **Clarity 18 breaking changes across many components** → Clarity has Clarity-specific schematics; run them and then audit remaining template errors manually. Prioritize high-usage components (data grids, modals, alerts, navigation).

- **`scripts` array removal from `application` builder** → Global scripts like `marked.min.js` and `prismjs` cannot be declared in `angular.json`'s `scripts` array. They must be imported in `main.ts` or a relevant module, or declared as `externalDependencies`.

- **Karma test runner compatibility** → Karma will likely require `@angular/build:karma` or the karma builder equivalent for Angular 21. Verify the test builder name in `angular.json` after running `ng update`.

- **Node.js version requirement** → Angular 21 requires Node 20 LTS or 22 LTS. Verify CI agent images and developer `.nvmrc` / `package.engines` field match.

## Migration Plan

1. **Verify Node.js** — Confirm CI and local environment run Node 20+ or 22+.
2. **Run `ng update`** — Execute in the order described in Decision 5 to apply official schematics.
3. **Upgrade Clarity manually** — `npm install @clr/angular@18.0.0 @clr/ui@18.0.0`.
4. **Remove `@clr/icons`** — Update `package.json`, `angular.json` (styles/scripts), and `shared.module.ts`.
5. **Migrate `angular.json`** — Switch to `application` builder; move global scripts to imports.
6. **Migrate `tsconfig.json`** — Update `moduleResolution`, `module`, remove `downlevelIteration`.
7. **Fix compilation errors** — Run `ng build` and address all TypeScript and template errors.
8. **Fix test errors** — Run `ng test` and fix Karma/Jasmine configuration issues.
9. **Smoke test locally** — Boot the dev server (`ng serve`) and verify core flows.
10. **CI green** — Ensure the full CI pipeline (build + unit tests + E2E) passes before merging.

**Rollback:** The upgrade is isolated to a single branch. If the upgrade proves too costly mid-stream, the branch is abandoned and Angular 19 / Clarity 17 remain on the main branch unchanged.

## Open Questions

- Does Clarity 18.0.0 publish official Angular schematics for automated migration, or is the upgrade entirely manual?
- Are there any Clarity 18 components whose APIs changed in ways that affect Harbor-specific customizations (e.g., custom data-grid renderers, custom modal management)?
- Does the `application` builder support the current `proxy.config.mjs` dev-server proxy configuration, or does the proxy config format need updating?
