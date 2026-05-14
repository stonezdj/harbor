## 1. Prerequisites

- [x] 1.1 Verify local Node.js version is 20 LTS or 22 LTS (`node --version`); update `.nvmrc` if it exists
- [x] 1.2 Check CI pipeline agent image / `engines` field in `package.json` and update Node.js version requirement if needed
- [x] 1.3 Create a dedicated feature branch for the upgrade (e.g., `feat/upgrade-angular21-clarity18`)

## 2. Run Angular Update Schematics

- [x] 2.1 Run `npx ng update @angular/core@21 @angular/cli@21` and accept all schematic changes
- [x] 2.2 Run `npx ng update @angular/cdk@21` to update CDK and apply its schematics
- [x] 2.3 Run `npx ng update @angular-eslint/schematics@21` to update ESLint tooling
- [x] 2.4 Verify `package.json` shows all `@angular/*` and `@angular-devkit/*` packages at `^21.0.0`
- [x] 2.5 Run `npm install` after schematic runs to ensure `package-lock.json` is consistent

## 3. Upgrade Clarity

- [x] 3.1 Run `npm install @clr/angular@18.0.0-beta.12 @clr/ui@18.0.0-beta.12 --save-exact` (18.0.0 stable not yet released; using latest beta)
- [x] 3.2 Check if Clarity 18 ships an `ng update` schematic; if so, run `npx ng update @clr/angular@18.0.0-beta.12` and apply changes
- [x] 3.3 Verify `package.json` reflects `@clr/angular: "18.0.0-beta.12"` and `@clr/ui: "18.0.0-beta.12"`

## 4. Remove `@clr/icons` and Migrate to `@cds/core`

- [x] 4.1 Remove `@clr/icons` from `package.json` (`npm uninstall @clr/icons`)
- [x] 4.2 Update `src/app/shared/shared.module.ts`: replace `@clr/icons` imports with `@cds/core/icon` equivalents using `ClarityIcons.addIcons(...allShapes)`
- [x] 4.3 Remove `node_modules/@clr/icons/clr-icons.min.css` from the `styles` array in `angular.json`
- [x] 4.4 Remove `node_modules/@clr/icons/clr-icons.min.js` from the `scripts` array in `angular.json`
- [x] 4.5 Confirm `@cds/core` is listed in `package.json` dependencies (it may arrive as a transitive dep of `@clr/angular`); add explicitly if missing

## 5. Migrate `angular.json` to the `application` Builder

- [x] 5.1 Change the `build` target `builder` from `@angular-devkit/build-angular:browser` to `@angular-devkit/build-angular:application`
- [x] 5.2 Rename the `main` option key to `browser` (value stays `src/main.ts`)
- [x] 5.3 Convert `polyfills` from a string to an array: `"polyfills": ["zone.js"]`
- [x] 5.4 Remove the `aot` option (the `application` builder is always AOT)
- [x] 5.5 Remove the `vendorChunk`, `namedChunks`, and `buildOptimizer` options (not supported by `application` builder)
- [x] 5.6 Move `scripts` entries (`marked.min.js`, `prism.js`, `prism-yaml.min.js`) to explicit imports at the top of `src/main.ts` or relevant lazy modules that use them
- [x] 5.7 Verify the lazy theme bundles (`dark-theme.scss`, `light-theme.scss` with `inject: false`) remain in the `styles` array — the `application` builder supports them
- [x] 5.8 Update the `serve` target `builder` to `@angular-devkit/build-angular:dev-server` if the schematic did not already do so; confirm `proxy.config.mjs` is still referenced
- [x] 5.9 Update the `test` target `builder` to `@angular-devkit/build-angular:karma` (or `@angular/build:karma` if Angular 21 renames it) and verify the karma config path is correct
- [x] 5.10 Verify the `extract-i18n` target builder is updated if the schematic did not do so

## 6. Update `tsconfig.json`

- [x] 6.1 Change `"moduleResolution": "node"` to `"moduleResolution": "bundler"` (done by ng update schematic)
- [x] 6.2 Change `"module": "es2020"` to `"module": "ES2022"`
- [x] 6.3 Remove `"downlevelIteration": true` (redundant for `target: ES2022`)
- [x] 6.4 Add `"esModuleInterop": true`
- [x] 6.5 Keep `"experimentalDecorators": true` (required for NgModule decorator syntax until standalone migration)
- [x] 6.6 Verify `tsconfig.app.json` and `tsconfig.spec.json` extend the root config correctly and do not override the changed options incorrectly

## 7. Fix Angular Compilation Errors

- [x] 7.1 Run `ng build` and capture all TypeScript and template errors to a log file
- [x] 7.2 Fix any removed or renamed Angular APIs (search release notes for Angular 20–21 breaking changes; common areas: `HttpClientModule` → `provideHttpClient`, removed `ComponentFactoryResolver`, lifecycle hook renames)
- [x] 7.3 Fix any template type-checking errors now surfaced by AOT-only compilation (previously hidden by `aot: false` in dev builds)
- [x] 7.4 Fix any `@NgModule` import/export issues caused by Clarity 18 module restructuring
- [x] 7.5 Resolve any peer-dependency or `allowedCommonJsDependencies` warnings introduced by new package versions
- [x] 7.6 Run `ng build --configuration production` and fix any production-only errors (e.g., tree-shaking side-effects)

## 8. Fix Clarity 18 Template Breaking Changes

- [x] 8.1 Audit Clarity 18.0.0 changelog / migration guide for removed or renamed component inputs, outputs, and selectors
- [x] 8.2 Find and fix all uses of renamed/removed Clarity selectors in HTML templates (e.g., data grid column inputs, alert variants, button types)
- [x] 8.3 Update any Clarity component class names or service APIs that changed in v18 (e.g., modal service, signaling APIs)
- [x] 8.4 Verify that `ClrModule` (or individual Clarity module imports in `shared.module.ts`) still exports all needed components for the app's NgModules
- [ ] 8.5 Run a visual smoke-test of the main UI areas (data grids, modals, navigation, alerts, forms) to detect layout regressions

## 9. Fix Unit Tests

- [x] 9.1 Run `ng test --watch=false` and capture test failure output
- [x] 9.2 Fix Karma/Jasmine configuration issues (e.g., outdated test builder options in `angular.json`)
- [x] 9.3 Update test bed setup (`TestBed.configureTestingModule`) for components that relied on Angular APIs changed in v20–21
- [x] 9.4 Fix Clarity-related spec failures caused by component API changes in v18 (update mock inputs/outputs, fix selector queries)
- [x] 9.5 Confirm all tests pass: `ng test --watch=false` exits with code 0

## 10. Verify Dev Server and Proxy

- [ ] 10.1 Run `ng serve` and confirm the dev server starts without errors on `http://localhost:4200`
- [ ] 10.2 Verify the proxy config (`proxy.config.mjs`) correctly forwards `/api/**` to the backend; test at least one proxied API call in the browser
- [ ] 10.3 Confirm hot-reload / HMR still works after a source file change

## 11. CI and Build Pipeline

- [ ] 11.1 Check whether any Dockerfile or CI YAML script copies from `dist/` directly; if the `application` builder changed the output path to `dist/harbor-portal/browser/`, update those references
- [ ] 11.2 Update CI Node.js version pin to 20 LTS or 22 LTS in all relevant pipeline files
- [ ] 11.3 Run the full CI pipeline (build + lint + unit tests) and confirm green status

## 12. Final Verification

- [x] 12.1 Run `ng build --configuration production` one final time and confirm zero errors and zero warnings
- [x] 12.2 Run `ng test --watch=false` and confirm all tests pass
- [ ] 12.3 Perform a manual smoke-test of the portal covering: login, project list, artifact list, replication rules, system settings, and user management
- [ ] 12.4 Review and merge the PR after CI passes
