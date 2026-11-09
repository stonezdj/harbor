# Harbor Playwright tests

Playwright (TypeScript) ports of the Robot Framework UI cases in `tests/robot-cases`.

| Spec | Ported from |
| --- | --- |
| `tests/db.spec.ts` | `robot-cases/Group1-Nightly/DB.robot` |

Page helpers live in `pages/harbor.ts`; their selectors mirror `tests/resources/Harbor-Pages/*_Elements.robot`.

## Run

```bash
cd tests/playwright
npm install
npx playwright install chromium

# HARBOR_URL (or IP, like the robot ${ip}) and the admin password
HARBOR_URL=https://10.0.0.1 HARBOR_PASSWORD=Harbor12345 npm run test:db

# a single case, by robot tag
npx playwright test --grep @update_password
```

The DB cases start from a fresh Harbor in database auth mode with self-registration **disabled**, and
run serially because they toggle that setting — the same assumptions as `DB.robot`.
