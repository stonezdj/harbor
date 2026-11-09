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

// Playwright port of tests/robot-cases/Group1-Nightly/DB.robot (Harbor BATs, DB auth mode).
//
// The cases share global state (the self-registration setting) and expect to run
// in file order, exactly like the robot suite, so the describe block is serial.

import { test } from '@playwright/test';
import { HarborPage, HARBOR_ADMIN, HARBOR_PASSWORD, NewUser, uniqueSuffix } from '../pages/harbor';

function newUser(prefix: string, d: string, overrides: Partial<NewUser> = {}): NewUser {
  return {
    username: `${prefix}${d}`,
    email: `${prefix}${d}@harbortest.com`,
    realname: 'harbortest',
    password: 'Test1@34',
    comment: 'harbortest',
    ...overrides,
  };
}

test.describe.configure({ mode: 'serial' });

test.describe('DB - Nightly', { tag: '@Nightly' }, () => {
  let harbor: HarborPage;

  test.beforeEach(async ({ page }) => {
    harbor = new HarborPage(page);
  });

  test('Create An New User', { tag: '@create_new_user' }, async () => {
    const d = uniqueSuffix();
    await harbor.signIn(HARBOR_ADMIN, HARBOR_PASSWORD);
    await harbor.switchToConfigure();
    await harbor.selfRegShouldBeDisabled();
    await harbor.enableSelfReg();
    await harbor.logout();

    await harbor.createNewUser(newUser('tester', d));
  });

  test('Update User Comment', { tag: '@update_user_comment' }, async () => {
    const d = uniqueSuffix();
    await harbor.createNewUser(newUser('tester', d));
    await harbor.updateUserComment('Test12#4');
    await harbor.logout();
  });

  test('Update Password', { tag: '@update_password' }, async () => {
    const d = uniqueSuffix();
    const user = newUser('tester', d);
    await harbor.createNewUser(user);
    await harbor.changePassword('Test1@34', 'Test12#4');
    await harbor.logout();
    await harbor.signIn(user.username, 'Test12#4');
  });

  test('Delete Multi User', { tag: '@delete_multi_user' }, async () => {
    const d = uniqueSuffix();
    for (const p of ['a', 'b', 'c']) {
      await harbor.createNewUser({
        username: `delete${p}${d}`,
        email: `test${p}${d}@harbortest.com`,
        realname: `test${d}`,
        password: 'Test1@34',
        comment: 'harbor',
      });
      await harbor.logout();
    }
    await harbor.signIn(HARBOR_ADMIN, HARBOR_PASSWORD);
    await harbor.switchToUserTab();
    await harbor.filterObject('delete');
    await harbor.multiDeleteUser(`deletea${d}`, `deleteb${d}`, `deletec${d}`);
    await harbor.deleteSuccess(`deletea${d}`, `deleteb${d}`, `deletec${d}`);
  });

  test('Edit Self-Registration', { tag: '@edit_self_registration' }, async () => {
    await harbor.signIn(HARBOR_ADMIN, HARBOR_PASSWORD);
    await harbor.disableSelfReg();
    await harbor.logout();

    await harbor.signUpShouldNotDisplay();

    await harbor.signIn(HARBOR_ADMIN, HARBOR_PASSWORD);
    await harbor.switchToConfigure();
    await harbor.selfRegShouldBeDisabled();

    // Restore setting
    await harbor.enableSelfReg();
  });

  test('Admin Add New Users', { tag: '@admin_add_new_users' }, async () => {
    const d = uniqueSuffix();
    await harbor.signIn(HARBOR_ADMIN, HARBOR_PASSWORD);

    // can add a new user when self-reg is enabled
    await harbor.switchToConfigure();
    await harbor.selfRegShouldBeEnabled();
    await harbor.switchToUserTab();
    await harbor.addNewUser(newUser('tester', d));

    // can add a new user when self-reg is disabled
    await harbor.switchToConfigure();
    await harbor.selfRegShouldBeEnabled();
    await harbor.disableSelfReg();
    await harbor.selfRegShouldBeDisabled();
    await harbor.switchToUserTab();
    await harbor.addNewUser(newUser('tester2', d, { realname: 'harbortest2', comment: 'harbortest2' }));
  });
});
