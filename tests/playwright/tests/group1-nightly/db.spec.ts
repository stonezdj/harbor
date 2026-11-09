// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Ported from tests/robot-cases/Group1-Nightly/DB.robot
import { expect, test } from '@playwright/test';
import {
    HARBOR_ADMIN,
    HARBOR_PASSWORD,
    addNewUser,
    changePassword,
    createNewUser,
    deleteSuccess,
    disableSelfReg,
    enableSelfReg,
    filterObject,
    logout,
    multiDeleteUser,
    selfRegShouldBeDisabled,
    selfRegShouldBeEnabled,
    signIn,
    switchToConfigure,
    switchToUserTag,
    uniqueSuffix,
    updateUserComment,
} from '../../helpers/harbor';

const userFor = (username: string, extra: Partial<Record<string, string>> = {}) => ({
    username,
    email: `${username}@harbortest.com`,
    realname: 'harbortest',
    newPassword: 'Test1@34',
    comment: 'harbortest',
    ...extra,
});

test.describe('Group1-Nightly DB', () => {
    test('Create An New User @create_new_user', async ({ page }) => {
        const d = uniqueSuffix();
        await page.goto('/');
        await signIn(page, HARBOR_ADMIN, HARBOR_PASSWORD);
        await switchToConfigure(page);
        await selfRegShouldBeDisabled(page);
        await enableSelfReg(page);
        await logout(page);

        await createNewUser(page, userFor(`tester${d}`));
    });

    test('Update User Comment @update_user_comment', async ({ page }) => {
        const d = uniqueSuffix();
        await createNewUser(page, userFor(`tester${d}`));
        await updateUserComment(page, 'Test12#4');
        await logout(page);
    });

    test('Update Password @update_password', async ({ page }) => {
        const d = uniqueSuffix();
        await createNewUser(page, userFor(`tester${d}`));
        await changePassword(page, 'Test1@34', 'Test12#4');
        await logout(page);
        await signIn(page, `tester${d}`, 'Test12#4');
    });

    test('Delete Multi User @delete_multi_user', async ({ page }) => {
        const d = uniqueSuffix();
        for (const p of ['a', 'b', 'c']) {
            await createNewUser(page, {
                username: `delete${p}${d}`,
                email: `test${p}${d}@harbortest.com`,
                realname: `test${d}`,
                newPassword: 'Test1@34',
                comment: 'harbor',
            });
            await logout(page);
        }
        await signIn(page, 'admin', 'Harbor12345');
        await switchToUserTag(page);
        await filterObject(page, 'delete');
        await multiDeleteUser(page, 'deletea', 'deleteb', 'deletec');
        await deleteSuccess(page, 'deletea', 'deleteb', 'deletec');
    });

    test('Edit Self-Registration @edit_self_registration', async ({ page }) => {
        await page.goto('/');
        await signIn(page, HARBOR_ADMIN, HARBOR_PASSWORD);
        await disableSelfReg(page);
        await logout(page);

        // Sign-up link must not be shown while self-registration is off.
        await expect(page.locator('a.signup')).toHaveCount(0);

        await signIn(page, HARBOR_ADMIN, HARBOR_PASSWORD);
        await switchToConfigure(page);
        await selfRegShouldBeDisabled(page);

        // Restore setting
        await enableSelfReg(page);
    });

    test('Admin Add New Users @admin_add_new_users', async ({ page }) => {
        const d = uniqueSuffix();
        await page.goto('/');
        await signIn(page, HARBOR_ADMIN, HARBOR_PASSWORD);

        // can add a new user when self-reg is enabled
        await switchToConfigure(page);
        await selfRegShouldBeEnabled(page);
        await switchToUserTag(page);
        await addNewUser(page, userFor(`tester${d}`));

        // can add a new user when self-reg is disabled
        await switchToConfigure(page);
        await selfRegShouldBeEnabled(page);
        await disableSelfReg(page);
        await selfRegShouldBeDisabled(page);
        await switchToUserTag(page);
        await addNewUser(
            page,
            userFor(`tester2${d}`, { realname: 'harbortest2', comment: 'harbortest2' })
        );
    });
});
