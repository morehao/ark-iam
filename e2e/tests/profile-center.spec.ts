import { test, expect } from '@playwright/test';
import { adminDirectLogin } from '../helpers/oidc-helpers';

test.describe('个人中心', () => {
  test('登录后打开个人中心：个人信息与会话接口正常返回', async ({ page }) => {
    await adminDirectLogin(page);

    // 点击头像打开下拉菜单，进入个人中心
    const avatar = page.locator('.ant-avatar');
    await expect(avatar).toBeVisible({ timeout: 10000 });
    await avatar.click();
    const profileItem = page.locator('.ant-dropdown-menu-item', { hasText: '个人中心' });
    await expect(profileItem).toBeVisible({ timeout: 5000 });

    // 在触发请求前注册响应监听，验证个人中心依赖的业务接口非 404（baseURL 含 /v1/iam）
    const personDetailPromise = page.waitForResponse((r) => r.url().includes('/person/detail'), { timeout: 15000 });
    const sessionsPromise = page.waitForResponse(
      (r) => r.url().includes('/user/sessions') && r.request().method() === 'GET',
      { timeout: 15000 },
    );

    await profileItem.click();

    // 个人中心 Modal 打开，个人信息应展示真实数据（seed admin:
    //   person.name = 系统管理员，username = admin），证明接口成功返回
    await expect(page.locator('.ant-modal')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-descriptions')).toContainText('系统管理员', { timeout: 15000 });
    await expect(page.locator('.ant-descriptions')).toContainText('admin');

    const personResp = await personDetailPromise;
    const sessionsResp = await sessionsPromise;
    expect(personResp.ok()).toBe(true);
    expect(sessionsResp.ok()).toBe(true);

    // 切换到"会话管理" tab，会话表格应正常渲染（仅验证加载成功，不触发撤销等写操作）
    await page.locator('.ant-tabs-tab', { hasText: '会话管理' }).click();
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 10000 });
  });
});
