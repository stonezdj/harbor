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

// Page helpers ported from tests/resources/Harbor-Pages/*.robot.
// Selectors are kept identical to the *_Elements.robot files so both suites
// break (and get fixed) together.

import { expect, Page } from '@playwright/test';

export const HARBOR_ADMIN = process.env.HARBOR_ADMIN || 'admin';
export const HARBOR_PASSWORD = process.env.HARBOR_PASSWORD || 'Harbor12345';

const sel = {
  // HomePage_Elements.robot
  harborTitle: "xpath=//span[contains(., 'Harbor')]",
  loginName: '#login_username',
  loginPwd: '#login_password',
  loginBtn: '#log_in',
  signUpForAccount: "xpath=//a[normalize-space()='Sign up for an account']",
  signUpButton: 'a.signup',
  username: '#username',
  email: '#email',
  realname: '#realname',
  newPassword: '#newPassword',
  confirmPassword: '#confirmPassword',
  comment: '#comment',
  signUp: '#sign-up',
  // UserProfile_Elements.robot
  headUser: "xpath=//clr-dropdown//button//clr-icon[@shape='user']",
  changePassword: "xpath=(//a[normalize-space()='Change Password'])[1]",
  userProfile: "xpath=(//a[normalize-space()='User Profile'])[1]",
  oldPassword: '#oldPassword',
  reNewPassword: '#reNewPassword',
  changePasswordConfirm: 'xpath=//password-setting/clr-modal//button[2]',
  userProfileConfirm: 'xpath=//account-settings-modal/clr-modal//button[2]',
  signInTitle: "xpath=//sign-in//form//*[@class='title']",
  accountComments: '#account_settings_comments',
  // Configuration_Elements.robot
  configuration: "xpath=//clr-main-container//clr-vertical-nav//a[contains(.,' Configuration ')]",
  selfReg: '#selfReg',
  configAuthSave: '#config_auth_save',
  // Logs_Elements.robot
  logs: "xpath=//clr-main-container//clr-vertical-nav//a[contains(.,'Logs')]",
  // Administration-Users_Elements.robot
  usersTab: "xpath=//clr-vertical-nav-group-children/a[contains(.,'Users')]",
  addNewUser: '#add-new-user',
  saveNewUser: '#save-button',
  // ToolKit / Public elements
  filterIcon: 'hbr-filter clr-icon',
  filterInput: 'hbr-filter input',
  memberAction: '#member-action',
  deleteUser: '#deleteUser',
  modalDeleteBtn: "xpath=//clr-modal//button[contains(.,'DELETE')]",
};

/** Same idea as robot `Get Current Date result_format=%m%s`: a unique suffix per call. */
export function uniqueSuffix(): string {
  return `${Date.now()}${Math.floor(Math.random() * 1000)}`;
}

export interface NewUser {
  username: string;
  email: string;
  realname: string;
  password: string;
  comment: string;
}

export class HarborPage {
  constructor(readonly page: Page) {}

  // ---------- HomePage.robot ----------

  async signIn(user: string = HARBOR_ADMIN, pw: string = HARBOR_PASSWORD) {
    await this.page.goto('/');
    await expect(this.page.locator(sel.harborTitle).first()).toBeVisible();
    await this.page.locator(sel.loginName).fill(user);
    await this.page.locator(sel.loginPwd).fill(pw);
    await this.page.locator(sel.loginBtn).click();
    await expect(this.page.locator(`xpath=//span[contains(., '${user}')]`).first()).toBeVisible();
  }

  async signUpShouldNotDisplay() {
    await expect(this.page.locator(sel.signUpButton)).toHaveCount(0);
  }

  /** Self-register a user from the sign-in page, then log in as that user. */
  async createNewUser(u: NewUser) {
    await this.page.goto('/');
    await expect(this.page.locator(sel.harborTitle).first()).toBeVisible();
    await this.page.locator(sel.signUpForAccount).click();
    await this.fillUserForm(u);
    await this.page.locator(sel.signUp).click();
    await expect(this.page.locator(sel.signUp)).toHaveCount(0);
    await this.page.locator(sel.loginName).fill(u.username);
    await this.page.locator(sel.loginPwd).fill(u.password);
    await this.page.locator(sel.loginBtn).click();
    await expect(this.page.locator(sel.loginBtn)).toHaveCount(0);
    await expect(this.page.locator(`xpath=//span[contains(., '${u.username}')]`).first()).toBeVisible();
  }

  private async fillUserForm(u: NewUser) {
    await this.page.locator(sel.username).fill(u.username);
    await this.page.locator(sel.email).fill(u.email);
    await this.page.locator(sel.realname).fill(u.realname);
    await this.page.locator(sel.newPassword).fill(u.password);
    await this.page.locator(sel.confirmPassword).fill(u.password);
    await this.page.locator(sel.comment).fill(u.comment);
  }

  // ---------- UserProfile.robot ----------

  async logout() {
    await this.page.locator(sel.headUser).click();
    await this.page.getByRole('link', { name: 'Log Out' }).click();
    await expect(this.page.locator(sel.signInTitle)).toBeVisible();
  }

  async changePassword(current: string, next: string) {
    await this.page.locator(sel.headUser).click();
    await this.page.locator(sel.changePassword).click();
    await this.page.locator(sel.oldPassword).fill(current);
    await this.page.locator(sel.newPassword).fill(next);
    await this.page.locator(sel.reNewPassword).fill(next);
    await this.page.locator(sel.changePasswordConfirm).click();
    await expect(this.page.locator('password-setting clr-modal .modal')).toBeHidden();
    await this.page.locator(sel.logs).click();
  }

  async updateUserComment(comment: string) {
    await this.page.locator(sel.headUser).click();
    await this.page.locator(sel.userProfile).click();
    await this.page.locator(sel.accountComments).fill(comment);
    await this.page.locator(sel.userProfileConfirm).click();
    await expect(this.page.locator('account-settings-modal clr-modal .modal')).toBeHidden();
  }

  // ---------- Configuration.robot ----------

  async switchToConfigure() {
    await this.page.locator(sel.configuration).click();
    await expect(this.page.locator(sel.selfReg)).toBeAttached();
  }

  async selfRegShouldBeEnabled() {
    await expect(this.page.locator(sel.selfReg)).toBeChecked();
  }

  async selfRegShouldBeDisabled() {
    await expect(this.page.locator(sel.selfReg)).not.toBeChecked();
  }

  /** Assumes the Configuration page is already open (same as the robot keyword). */
  async enableSelfReg() {
    await this.page.locator(sel.selfReg).check({ force: true });
    await this.selfRegShouldBeEnabled();
    await this.saveAuthConfig();
  }

  /** Opens the Configuration page first (same as the robot keyword). */
  async disableSelfReg() {
    await this.page.locator(sel.configuration).click();
    await this.page.locator(sel.selfReg).uncheck({ force: true });
    await this.selfRegShouldBeDisabled();
    await this.saveAuthConfig();
  }

  private async saveAuthConfig() {
    const save = this.page.locator(sel.configAuthSave);
    await expect(save).toBeEnabled();
    await save.click();
    await expect(this.page.getByText('Configuration has been successfully saved.')).toBeVisible();
  }

  // ---------- Administration-Users.robot ----------

  async switchToUserTab() {
    await this.page.locator(sel.usersTab).click();
    await expect(this.page.locator('harbor-user')).toBeVisible();
  }

  /** Admin creates a user from Administration > Users. */
  async addNewUser(u: NewUser) {
    await this.page.locator(sel.addNewUser).click();
    await this.fillUserForm(u);
    await this.page.locator(sel.saveNewUser).click();
    await expect(this.page.locator(sel.saveNewUser)).toHaveCount(0);
    await expect(
      this.page.locator(`xpath=//harbor-user//clr-dg-row//clr-dg-cell[contains(., '${u.username}')]`).first(),
    ).toBeVisible();
  }

  // ---------- ToolKit.robot ----------

  async filterObject(keyword: string) {
    await this.page.locator(sel.filterIcon).first().click();
    const input = this.page.locator(sel.filterInput).first();
    await expect(input).toBeEditable();
    await input.fill(keyword);
    // The filter is debounced; wait for the grid to settle on matching rows.
    await expect(this.page.locator('clr-dg-row').first()).toContainText(keyword);
  }

  async multiDeleteUser(...names: string[]) {
    for (const name of names) {
      await this.page
        .locator(`xpath=//clr-dg-row[contains(.,'${name}')]//label[contains(@class,'clr-control-label')]`)
        .first()
        .click();
    }
    await this.page.locator(sel.memberAction).click();
    await this.page.locator(sel.deleteUser).click();
    const confirm = this.page.locator(sel.modalDeleteBtn);
    await confirm.click();
    await expect(confirm).toHaveCount(0);
  }

  async deleteSuccess(...names: string[]) {
    for (const name of names) {
      await expect(
        this.page
          .locator(`xpath=//*[@id='contentAll']//div[contains(.,'${name}')]/../div/clr-icon[@shape='success-standard']`)
          .first(),
      ).toBeAttached();
    }
  }
}
