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

    // 在触发请求前注册响应监听，验证个人中心依赖的业务接口非 404（baseURL 含 /v1）。
    //
    // 路径已按"当前用户资源"改造更新：`/person/detail` → `GET /v1/auth/me`、
    // `/user/sessions` → `GET /v1/auth/me/sessions`（见 AGENTS.md「当前用户资源」）。
    // 旧路径是更早一次重构前的写法，早已不存在，因此这两条等待**永远**等不到响应——
    // 个人中心页面本身是好的，坏的是用例里的 URL 匹配。
    // 注意 `/auth/me` 是 `/auth/me/sessions` 的前缀，两个匹配器必须各自收紧到"路径末尾"，
    // 否则第一个会把会话请求也算进去，断言就失去了各自的意义。
    const personDetailPromise = page.waitForResponse(
      (r) => /\/v1\/auth\/me(\?|$)/.test(new URL(r.url()).pathname + new URL(r.url()).search),
      { timeout: 15000 },
    );
    const sessionsPromise = page.waitForResponse(
      (r) => new URL(r.url()).pathname.endsWith('/v1/auth/me/sessions') && r.request().method() === 'GET',
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
