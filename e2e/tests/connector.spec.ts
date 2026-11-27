import { test, expect } from '@playwright/test';
import { adminDirectLogin } from '../helpers/oidc-helpers';

/**
 * 身份连接器（平台管理后台 /connector）端到端用例。
 *
 * 归属改造后：连接器是**平台级全局对象**，页面挂平台管理台（:4001），管理面走
 * `/v1/platform/connectors*`（平台管理员跨租户），运行时（authorize/callback）仍在 auth。
 * 编辑能力的「默认落租户」= 自动创建用户时用户默认落在的租户（default_tenant_id）。
 *
 * 前置：globalSetup 已起好 gateway(:8100) + login-web(:4000) + platform-admin-web(:4001)，
 * 并完成/复用系统初始化（管理员 admin / Admin123）。平台控制台侧栏含「身份连接器」菜单
 * （内置菜单挂 platform_admin 应用 grp-platform 组）。
 *
 * 数据：连接器为全局对象，用例用唯一 displayName 建、断言、改、删，避免污染与重复。
 */

const CONNECTOR_PATH = '/connector';

function uniqueName(): { name: string; displayName: string } {
  // 标识名需匹配 ^[a-z][a-z0-9-]*$；显示名用时间戳保证唯一，避免多轮运行冲突。
  const stamp = Date.now().toString();
  return {
    name: `e2e-google-${stamp}`,
    displayName: `E2E Google ${stamp}`,
  };
}

test.describe('身份连接器管理（平台端）', () => {
  test('从平台侧栏进入连接器页并完成 建-查-改-测-删 全流程', async ({ page }) => {
    await adminDirectLogin(page);

    // 1) 从侧栏菜单点入：连接器挂在「平台管理」目录组下（platform_admin 应用 grp-platform），
    //    先展开该组，再点叶子「身份连接器」（验证平台菜单已下发 + 路由可达）。
    await page.getByText('平台管理', { exact: true }).first().click();
    await page.getByText('身份连接器', { exact: true }).first().click();
    await expect(page).toHaveURL(new RegExp(`${CONNECTOR_PATH}`));
    await expect(page.getByText('身份连接器', { exact: true }).first()).toBeVisible();
    // 页面描述：平台级统一接入（不再是「本租户」）
    await expect(page.getByText('平台级身份提供商', { exact: false }).first()).toBeVisible({ timeout: 15000 });

    const uniq = uniqueName();

    // 2) 新建：进入 5 步向导
    await page.getByRole('button', { name: /新建连接器/ }).click();
    await expect(page.getByText('选择提供商', { exact: true }).first()).toBeVisible();

    // 第 1 步：选 Google 工厂（工厂卡按 Google / GitHub / Microsoft 顺序渲染，取第一个「选择」）
    await expect(page.locator('strong', { hasText: 'Google' }).first()).toBeVisible();
    await page.getByRole('button', { name: /选\s*择/ }).first().click();

    // 第 2 步：基本信息
    await expect(page.getByText('已选择提供商', { exact: false })).toBeVisible();
    const nameInput = page.getByPlaceholder('如 google-workspace');
    const displayInput = page.getByPlaceholder('如 Google Workspace');
    await nameInput.fill(uniq.name);
    await displayInput.fill(uniq.displayName);
    await page.getByRole('button', { name: '下一步' }).click();

    // 第 3 步：协议配置（Google=OIDC：issuer/clientId/clientSecret/redirectUri 必填）
    await page.getByPlaceholder('https://accounts.google.com').fill('https://accounts.google.com');
    await page.getByPlaceholder('由身份提供商分配').first().fill(`cid-${Date.now()}`);
    await page.getByPlaceholder('由身份提供商分配').last().fill('secret-abc-123');
    await page.getByPlaceholder('https://iam.example.com/oidc/callback').fill('https://iam.example.com/oidc/callback');
    await page.getByRole('button', { name: '下一步' }).click();

    // 第 4 步：声明映射（留空用默认）
    await page.getByRole('button', { name: '下一步' }).click();

    // 第 5 步：域策略与策略开关 → 打开「自动创建用户」并选择「默认落租户」（必填），再创建。
    //   向导是全屏覆盖层（fixed div，非 antd Modal），故用页面级可访问名精确定位。
    await expect(page.getByText('策略开关', { exact: false })).toBeVisible();
    // 打开「自动创建用户」开关（按可访问名精确定位，避免误点其它开关）
    await page.getByRole('switch', { name: /自动创建用户/ }).click();
    // 「默认落租户」下拉：按 combobox 可访问名定位，选第一项（平台租户列表）
    const defaultTenantCombo = page.getByRole('combobox', { name: /默认落租户/ });
    await expect(defaultTenantCombo).toBeVisible();
    await defaultTenantCombo.click();
    const firstTenantOption = page.locator('.ant-select-item-option').first();
    await expect(firstTenantOption).toBeVisible({ timeout: 10000 });
    const chosenTenant = (await firstTenantOption.textContent())?.trim() ?? '';
    await firstTenantOption.click();

    await page.getByRole('button', { name: '创建连接器' }).click();

    // 3) 列表出现新建行，且「默认落租户」列展示所选租户名
    const createdRow = page.locator('tr').filter({ hasText: uniq.displayName }).first();
    await expect(page.getByText(uniq.displayName, { exact: false }).first()).toBeVisible({ timeout: 20000 });
    if (chosenTenant) {
      await expect(createdRow.getByText(chosenTenant, { exact: false })).toBeVisible({ timeout: 10000 });
    }

    // 4) 打开详情：secret 应为脱敏占位符 + 有「已加密存储」标记
    await page.getByText(uniq.displayName, { exact: false }).first().click();
    const drawer = page.getByRole('dialog').filter({ hasText: '连接器详情' });
    await expect(drawer).toBeVisible();
    await expect(drawer.getByText('已加密存储', { exact: false })).toBeVisible();
    await expect(drawer.getByText('******', { exact: false })).toBeVisible();
    // 关闭详情（antd Drawer 关闭按钮）
    await drawer.locator('.ant-drawer-close').click();

    // 5) 编辑：仅改显示名，secret 留空应保留
    const row = page.locator('tr').filter({ hasText: uniq.displayName }).first();
    await row.getByRole('button', { name: /编\s*辑/ }).click();
    // 向导直接进入第 2 步（编辑态），改显示名
    const newDisplay = `${uniq.displayName} 改`;
    const editDisplay = page.getByPlaceholder('如 Google Workspace');
    await editDisplay.fill(newDisplay);
    // 后续步骤「下一步」×3 到提交
    await page.getByRole('button', { name: '下一步' }).click(); // -> step3
    await page.getByRole('button', { name: '下一步' }).click(); // -> step4
    await page.getByRole('button', { name: '下一步' }).click(); // -> step5
    await page.getByRole('button', { name: '保存修改' }).click();

    await expect(page.getByText(newDisplay, { exact: false }).first()).toBeVisible({ timeout: 20000 });

    // 连通性测试：连接器页当前无独立「测试」操作（操作列仅 编辑/删除），此步移除。

    // 7) 删除（Popconfirm 二次确认）
    const delRow = page.locator('tr').filter({ hasText: newDisplay }).first();
    await delRow.getByRole('button', { name: /删\s*除/ }).click();
    await page.getByRole('button', { name: /确\s*定/ }).click();
    await expect(page.getByText(newDisplay, { exact: false })).toHaveCount(0, { timeout: 20000 });
  });

  test('列表页可切换状态筛选与按名称搜索', async ({ page }) => {
    await adminDirectLogin(page);
    // 连接器在「平台管理」目录组下，先展开再进入。
    await page.getByText('平台管理', { exact: true }).first().click();
    await page.getByText('身份连接器', { exact: true }).first().click();
    await expect(page).toHaveURL(new RegExp(`${CONNECTOR_PATH}`));

    // 状态分段切换不报错
    await page.getByText('启用', { exact: true }).first().click();
    await page.getByText('全部', { exact: true }).first().click();

    // 搜索框可用
    const search = page.getByPlaceholder('按显示名 / 标识名搜索');
    await expect(search).toBeVisible();
    await search.fill('nonexistent-xyz');
    await search.press('Enter');
    // 无匹配也不应崩溃（表格空态）
    await expect(page.getByText('身份连接器', { exact: true }).first()).toBeVisible();
  });
});
