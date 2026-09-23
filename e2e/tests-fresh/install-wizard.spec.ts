import { test, expect } from '@playwright/test';

// AC-2 的浏览器路径：全新库上，运维用**页面**走完三步向导 → 提交 → 看到 Report
// → 立刻用刚创建的账号登录平台控制台。
//
// 与 e2e/tests/install.spec.ts（主配置）的分工：那个 spec 验证的是"已初始化后"的契约
// （status 如实报告 + 自锁 409/107000），它的库由 globalSetup 经**接口**初始化；
// 本用例补的是**页面**这条路径——D-P12 的验收缺口正在于此。
//
// 前置：网关跑在 :8100 且指向一个**全新的**库（BOOTSTRAP_TOKEN=e2e-bootstrap-token）；
// login-web :4000 / platform-admin-web :4001 已在运行。
const TENANT_NAME = '向导验证平台';
const ADMIN_USERNAME = 'wizardadmin';
const ADMIN_PASSWORD = 'Wizard123';
const ADMIN_NAME = '向导管理员';
const ADMIN_EMAIL = 'wizard@example.com';

test('全新库：页面走完三步向导 → 展示 Report → 用该账号登录平台控制台', async ({ page }) => {
  await page.goto('/install', { waitUntil: 'domcontentloaded' });

  // 未初始化 -> 必须给出可提交的表单（而不是"已完成"）
  await expect(page.getByText('系统初始化', { exact: true }).first()).toBeVisible({ timeout: 15000 });

  // ---- 第①步：租户 ----
  await expect(page.locator('#install-tenant-name')).toBeVisible({ timeout: 15000 });
  await page.fill('#install-tenant-name', TENANT_NAME);
  await page.getByRole('button', { name: '下一步' }).click();

  // ---- 第②步：管理员 ----
  await expect(page.locator('#install-admin-username')).toBeVisible({ timeout: 10000 });
  await page.fill('#install-admin-username', ADMIN_USERNAME);
  await page.fill('#install-admin-password', ADMIN_PASSWORD);
  await page.fill('#install-admin-confirm', ADMIN_PASSWORD);
  await page.fill('#install-admin-name', ADMIN_NAME);
  await page.fill('#install-admin-email', ADMIN_EMAIL);
  await page.getByRole('button', { name: '下一步' }).click();

  // ---- 第③步：内置数据确认（回显前两步填的值）----
  await expect(page.locator('#install-bootstrap-token')).toBeVisible({ timeout: 10000 });
  await expect(page.locator('.install-summary')).toContainText(TENANT_NAME);
  await expect(page.locator('.install-summary')).toContainText(ADMIN_USERNAME);

  // 初始化令牌只在本次提交用（请求头 X-Bootstrap-Token），不写入浏览器存储
  await page.fill('#install-bootstrap-token', 'e2e-bootstrap-token');
  await page.getByRole('button', { name: '确认初始化' }).click();

  // ---- 提交成功：展示 Report ----
  await expect(page.getByText('初始化完成', { exact: true })).toBeVisible({ timeout: 30000 });
  // 注意 `.install-summary-line` 现在有两个（管理员账号 + 内置数据条数），取第一个
  await expect(page.locator('.install-summary-line').first()).toContainText(ADMIN_USERNAME);

  // Report 的条数与明细入口（本次唯一一次内置数据写入，必须让运维看得见）
  const reportCount = page.locator('.install-report .install-summary-line');
  await expect(reportCount).toContainText('共写入');
  await page.locator('.install-report summary').click();
  const reportItems = page.locator('.install-report li');
  const count = await reportItems.count();
  expect(count).toBeGreaterThan(0);
  // 逐条是 entity/key/action 三元组，且 action 为 created（首次引导只会新建）
  await expect(reportItems.first()).toContainText('created');
  const reportText = await page.locator('.install-report ul').innerText();
  for (const key of ['tenant', 'menu', 'application_client', 'role']) {
    expect(reportText).toContain(key);
  }

  // ---- 用刚创建的账号登录平台控制台（控制台入口来自服务端配置）----
  const adminLink = page.locator('.install-consoles a[href*="4001"]').first();
  await expect(adminLink).toBeVisible();
  // 控制台入口是 target="_blank"，会开新标签页——必须等新页面，不能等当前页跳转
  const [adminPage] = await Promise.all([
    page.context().waitForEvent('page'),
    adminLink.click(),
  ]);
  await adminPage.waitForLoadState('domcontentloaded');

  // 落到 login-web 的 OIDC 授权页，填**刚在向导里创建**的账号
  await adminPage.waitForURL((url) => url.port === '4000' && url.pathname === '/login', { timeout: 30000 });
  await adminPage.fill('#identifier', ADMIN_USERNAME);
  await adminPage.fill('#password', ADMIN_PASSWORD);
  await adminPage.click('button[type="submit"]');

  // 进入平台控制台仪表盘 = 初始化产物真的可用
  await adminPage.waitForURL((url) => url.port === '4001' && !url.pathname.includes('/auth/callback'), { timeout: 30000 });
  await expect(adminPage.getByText('仪表盘', { exact: true }).first()).toBeVisible({ timeout: 30000 });
  await expect(adminPage.getByText(ADMIN_NAME, { exact: false }).first()).toBeVisible({ timeout: 15000 });
});
