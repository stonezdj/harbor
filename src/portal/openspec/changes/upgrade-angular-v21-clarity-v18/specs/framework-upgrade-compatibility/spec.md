## ADDED Requirements

### Requirement: Portal builds successfully with Angular 21 and Clarity 18
The portal application SHALL compile without errors or warnings using Angular 21 and Clarity 18.0.0, including both development and production build configurations.

#### Scenario: Development build succeeds
- **WHEN** `ng build` is run against the upgraded codebase in development mode
- **THEN** the build completes with exit code 0 and produces output in the configured `outputPath`

#### Scenario: Production build succeeds
- **WHEN** `ng build --configuration production` is run
- **THEN** the build completes with exit code 0, AOT compilation succeeds, and all bundles are emitted

#### Scenario: TypeScript compilation has no errors
- **WHEN** `ng build` is invoked
- **THEN** the TypeScript compiler reports zero errors across all project source files and templates

---

### Requirement: All existing unit tests pass on Angular 21
The unit test suite SHALL continue to pass without modification to test logic after the framework upgrade.

#### Scenario: Karma test run succeeds
- **WHEN** `ng test --watch=false` is executed
- **THEN** all previously passing tests still pass and no new test failures are introduced by the framework upgrade

---

### Requirement: Clarity 18 UI components render correctly
All Clarity UI components used in the portal SHALL render and function identically to their Clarity 17 counterparts after upgrading to Clarity 18.0.0.

#### Scenario: Data grid renders and paginates
- **WHEN** a user navigates to a page containing a Clarity `clr-datagrid`
- **THEN** the grid displays rows, column headers, and pagination controls without visual regression

#### Scenario: Modal dialogs open and close
- **WHEN** a user triggers an action that opens a Clarity modal
- **THEN** the modal appears, the backdrop is rendered, and closing the modal (via button or backdrop click) dismisses it correctly

#### Scenario: Navigation sidebar renders
- **WHEN** the portal is loaded
- **THEN** the Clarity navigation sidebar displays all expected menu items and responds to hover and click events

---

### Requirement: Icons render correctly via `@cds/core`
All icons previously loaded via `@clr/icons` SHALL continue to render correctly after migrating to `@cds/core` icon registration.

#### Scenario: Core icon shapes are visible
- **WHEN** a page containing Clarity icons (e.g., `cds-icon`, `clr-icon`) is rendered
- **THEN** each icon displays its correct shape without broken/missing SVG

#### Scenario: Custom or extended icon shapes are registered
- **WHEN** the application bootstraps
- **THEN** all icon shapes previously registered via `ClarityIcons.add()` from `@clr/icons` are available and render correctly after migration to `@cds/core`

---

### Requirement: Dev server proxy configuration remains functional
The development server's proxy configuration SHALL route API requests correctly after migrating to the `application` builder.

#### Scenario: API requests are proxied in dev mode
- **WHEN** `ng serve` is run and the browser makes a request to `/api/**`
- **THEN** the request is forwarded to the configured backend origin as defined in `proxy.config.mjs`

---

### Requirement: Existing routing and lazy loading works
All route-based lazy-loaded NgModules SHALL continue to load on navigation after the upgrade.

#### Scenario: Lazy route loads successfully
- **WHEN** a user navigates to a route that uses `loadChildren` to lazy-load a module
- **THEN** the module is fetched and the route component is rendered without console errors
