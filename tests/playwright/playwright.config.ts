import { defineConfig } from '@playwright/test';

// Usage: HARBOR_IP=<ip> [HARBOR_PASSWORD=Harbor12345] npx playwright test
const ip = process.env.HARBOR_IP ?? 'localhost';

export default defineConfig({
    testDir: './tests',
    timeout: 120_000,
    expect: { timeout: 15_000 },
    // These tests toggle global settings (self-registration), so run serially.
    workers: 1,
    fullyParallel: false,
    reporter: [['list'], ['html', { open: 'never' }]],
    use: {
        baseURL: process.env.HARBOR_URL ?? `https://${ip}`,
        ignoreHTTPSErrors: true,
        viewport: { width: 1600, height: 900 },
        screenshot: 'only-on-failure',
        trace: 'retain-on-failure',
    },
});
