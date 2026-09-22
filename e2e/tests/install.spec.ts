import { test, expect } from '@playwright/test';
import { CONFIG } from '../config';

// 初始化入口（/install）的契约验证。
//
// 为什么必须在 e2e 层再测一遍（后端已有单测）：这里是**真实部署形态**——
// 真实网关、真实顺序（globalSetup 刚刚用它引导过），且验证的是"自锁"这个安全属性。
// 自锁失效（重复初始化被放行）意味着任何能访问 /install 的人都能改已有系统的管理员，
// 属于必须在最接近生产的环境里确认的一条。

const INSTALL_STATUS = `${CONFIG.installBaseURL}/install/status`;
const INSTALL_INITIALIZE = `${CONFIG.installBaseURL}/install/initialize`;

test('系统已完成初始化：status 如实报告且引导入口自锁', async ({ request }) => {
  // 1. 状态：globalSetup 已引导过，必须报告 initialized=true
  const status = await request.get(INSTALL_STATUS);
  expect(status.status()).toBe(200);
  const statusBody = await status.json();
  expect(statusBody.data.initialized).toBe(true);
  expect(statusBody.data.schemaReady).toBe(true);
  expect(statusBody.data.tokenRequired).toBe(true);

  // 2. 自锁：再次初始化必须 409 + InstallationAlreadyInitializedError(107000)
  const again = await request.post(INSTALL_INITIALIZE, {
    headers: { 'X-Bootstrap-Token': CONFIG.bootstrapToken },
    data: {
      adminUsername: 'attacker',
      adminPassword: 'Attacker123',
      adminEmail: 'attacker@example.com',
    },
  });
  expect(again.status()).toBe(409);
  const againBody = await again.json();
  expect(againBody.code).toBe(107000);

  // 3. 令牌校验在自锁之前：无令牌必须 401 而不是 409——
  //    顺序很重要，否则未认证调用方能从状态码判断"系统是否已初始化"。
  const noToken = await request.post(INSTALL_INITIALIZE, {
    headers: {},
    data: {
      adminUsername: 'attacker',
      adminPassword: 'Attacker123',
      adminEmail: 'attacker@example.com',
    },
  });
  expect(noToken.status()).toBe(401);
  expect((await noToken.json()).code).toBe(107001);

  // 4. 错误令牌同样 401（与"未携带"同码：不向未认证方泄露"你差一点就对了"）
  const badToken = await request.post(INSTALL_INITIALIZE, {
    headers: { 'X-Bootstrap-Token': 'wrong-token' },
    data: {
      adminUsername: 'attacker',
      adminPassword: 'Attacker123',
      adminEmail: 'attacker@example.com',
    },
  });
  expect(badToken.status()).toBe(401);
  expect((await badToken.json()).code).toBe(107001);
});

test('已初始化时 /install 页面展示"已完成"而不是可提交的表单', async ({ page }) => {
  // 只断言"页面上不存在可提交的初始化表单"这类**结构性**事实，
  // 不锁定具体文案（文案属前端可迭代内容，锁死会让每次改字都要改 e2e）。
  await page.goto('http://localhost:4000/install', { waitUntil: 'domcontentloaded', timeout: 20000 });
  // 初始化表单必然包含口令输入；已初始化页不应有它
  await expect(page.locator('input[type="password"]')).toHaveCount(0, { timeout: 15000 });
  // 页面应给出控制台入口（至少一个指向 4001/4002 的链接）
  const consoleLinks = page.locator('a[href*="4001"], a[href*="4002"]');
  await expect(consoleLinks.first()).toBeVisible({ timeout: 15000 });
});
