import { expect, Page } from '@playwright/test';

export const HARBOR_ADMIN = process.env.HARBOR_ADMIN ?? 'admin';
export const HARBOR_PASSWORD = process.env.HARBOR_PASSWORD ?? 'Harbor12345';

/** Equivalent of Robot's `Get Current Date  result_format=%m%s` (unique suffix). */
export function uniqueSuffix(): string {
    const d = new Date();
    return `${d.getMonth() + 1}${Math.floor(d.getTime() / 1000)}`;
}

export async function signIn(page: Page, user: string, password: string) {
    await page.goto('/');
    await page.locator('#login_username').fill(user);
    await page.locator('#login_password').fill(password);
    await page.locator('#log_in').click();
    await expect(page.locator('clr-header').getByText(user)).toBeVisible();
}

export async function logout(page: Page) {
    await page.locator('clr-dropdown button:has(clr-icon[shape="user"])').click();
    await page.getByRole('link', { name: 'Log Out' }).click();
    await expect(page.locator('sign-in form .title')).toBeVisible();
}

export interface NewUser {
    username: string;
    email: string;
    realname: string;
    newPassword: string;
    comment: string;
}

async function fillUserForm(page: Page, u: NewUser) {
    await page.locator('#username').fill(u.username);
    await page.locator('#email').fill(u.email);
    await page.locator('#realname').fill(u.realname);
    await page.locator('#newPassword').fill(u.newPassword);
    await page.locator('#confirmPassword').fill(u.newPassword);
    await page.locator('#comment').fill(u.comment);
}

/** Self-sign-up from the login page, then log in as the new user. */
export async function createNewUser(page: Page, u: NewUser) {
    await page.goto('/');
    await page.getByRole('link', { name: 'Sign up for an account' }).click();
    await fillUserForm(page, u);
    await page.locator('#sign-up').click();
    await expect(page.locator('#sign-up')).toHaveCount(0);
    await page.locator('#login_username').fill(u.username);
    await page.locator('#login_password').fill(u.newPassword);
    await page.locator('#log_in').click();
    await expect(page.locator('#log_in')).toHaveCount(0);
    await expect(page.locator('span', { hasText: u.username }).first()).toBeVisible();
}

/** Admin adds a user from Administration -> Users. */
export async function addNewUser(page: Page, u: NewUser) {
    await page.locator('#add-new-user').click();
    await fillUserForm(page, u);
    await page.locator('#save-button').click();
    await expect(page.locator('#save-button')).toHaveCount(0);
    await expect(
        page.locator('harbor-user clr-dg-row clr-dg-cell', { hasText: u.username })
    ).toBeVisible();
}

export async function switchToConfigure(page: Page) {
    await page
        .locator('clr-main-container clr-vertical-nav')
        .getByRole('link', { name: 'Configuration' })
        .click();
}

export async function switchToUserTag(page: Page) {
    await page.locator('clr-vertical-nav-group-children a', { hasText: 'Users' }).click();
}

const selfReg = (page: Page) => page.locator('#selfReg');

export async function selfRegShouldBeEnabled(page: Page) {
    await expect(selfReg(page)).toBeChecked();
}

export async function selfRegShouldBeDisabled(page: Page) {
    await expect(selfReg(page)).not.toBeChecked();
}

async function toggleSelfReg(page: Page, expectEnabled: boolean) {
    // The Clarity checkbox input is visually hidden; force the click like Robot's Mouse Down/Up.
    await selfReg(page).click({ force: true });
    if (expectEnabled) {
        await selfRegShouldBeEnabled(page);
    } else {
        await selfRegShouldBeDisabled(page);
    }
    await page.locator('#config_auth_save').click();
}

export async function enableSelfReg(page: Page) {
    await toggleSelfReg(page, true);
}

/** Navigates to Configuration first (matches the Robot keyword). */
export async function disableSelfReg(page: Page) {
    await switchToConfigure(page);
    await toggleSelfReg(page, false);
}

export async function updateUserComment(page: Page, comment: string) {
    await page.locator('clr-dropdown button:has(clr-icon[shape="user"])').click();
    await page.getByRole('link', { name: 'User Profile' }).first().click();
    await page.locator('#account_settings_comments').fill(comment);
    await page.locator('account-settings-modal clr-modal button').nth(1).click();
}

export async function changePassword(page: Page, current: string, next: string) {
    await page.locator('clr-dropdown button:has(clr-icon[shape="user"])').click();
    await page.getByRole('link', { name: 'Change Password' }).first().click();
    await page.locator('#oldPassword').fill(current);
    await page.locator('#newPassword').fill(next);
    await page.locator('#reNewPassword').fill(next);
    await page.locator('password-setting clr-modal button').nth(1).click();
}

export async function filterObject(page: Page, keyword: string) {
    await page.locator('hbr-filter clr-icon').first().click();
    const input = page.locator('hbr-filter input');
    await input.fill(keyword);
}

export async function multiDeleteUser(page: Page, ...users: string[]) {
    for (const u of users) {
        await page
            .locator('clr-dg-row', { hasText: u })
            .locator('label.clr-control-label')
            .first()
            .click();
    }
    await page.locator('#member-action').click();
    await page.locator('#deleteUser').click();
    const confirm = page.locator('clr-modal button', { hasText: /delete/i });
    await confirm.click();
    await expect(confirm).toHaveCount(0);
}

export async function deleteSuccess(page: Page, ...objs: string[]) {
    for (const o of objs) {
        await expect(
            page.locator(
                `#contentAll div:has-text("${o}") >> xpath=../div/clr-icon[@shape='success-standard']`
            ).first()
        ).toBeVisible();
    }
}
