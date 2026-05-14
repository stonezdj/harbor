## Why

The portal currently runs on Angular 19 and Clarity 17, both of which will reach end-of-active-support as newer major versions ship. Upgrading to Angular 21 and Clarity 18.0.0 keeps the project on supported releases, unlocks security patches, performance improvements (including stabilized Signals APIs and improved SSR/hydration), and reduces the risk of a painful multi-version migration in the future.

## What Changes

- Upgrade all `@angular/*` runtime packages (`core`, `common`, `forms`, `router`, `cdk`, `animations`, `localize`, `platform-browser`, `platform-browser-dynamic`) from `^19.2.x` to `^21.0.0`
- Upgrade Angular build tooling (`@angular/cli`, `@angular-devkit/build-angular`, `@angular/compiler`, `@angular/compiler-cli`) to `^21.0.0`
- Upgrade `@angular-eslint/*` packages from `^19.2.x` to `^21.0.0`
- Upgrade `@clr/angular` and `@clr/ui` from `17.12.4` to `18.0.0`
- Remove `@clr/icons` (deprecated in Clarity 17; fully removed in 18) and migrate to `@cds/core` icon imports
- Update `typescript` to the range required by Angular 21 (expected `~5.8` or `~5.9`)
- Update `zone.js` to the version required by Angular 21 if the current `^0.15.x` is insufficient
- Fix any breaking API changes introduced across Angular 20–21 and Clarity 18 (deprecated lifecycle hooks, removed `ModuleWithProviders` patterns, updated Clarity component inputs/outputs, etc.)
- Update `angular.json`, `tsconfig.json`, and build configuration as needed for builder API changes

## Capabilities

### New Capabilities

_(none — this is a dependency upgrade; no new user-facing features are introduced)_

### Modified Capabilities

_(none — existing functional requirements are unchanged; only internal implementation details change to comply with updated framework APIs)_

## Impact

- **All Angular source files**: components, directives, pipes, services, and guards may need adjustments for APIs deprecated in Angular 19–20 and removed in Angular 21
- **Clarity templates**: Clarity 18 introduces breaking changes to several component inputs/outputs and removes deprecated selectors; all affected templates must be updated
- **`@clr/icons` removal**: All icon imports must migrate from `@clr/icons` to `@cds/core/icon` and the `ClarityIcons.addIcons()` registration pattern
- **Build configuration**: `angular.json` builder options may change; `tsconfig.json` may need `target` or `lib` adjustments
- **Node.js runtime**: Angular 21 requires Node.js 20 LTS or 22 LTS; CI/CD and dev environment Node version must be verified
- **Unit and E2E tests**: Test utilities and mocks that depend on Angular internals may need updating
