import { test, expect, type APIRequestContext } from '@playwright/test';
import { CONFIG } from '../config';
import { rp1Login } from '../helpers/oidc-helpers';

/**
 * 租户菜单管理 E2E（菜单维护权下放到归属租户后的功能验证）。
 *
 * 背景：菜单写能力从平台端下放到「应用归属租户」，入口在租户控制台 /menu
 * （tenant-admin-web :4002），后端 /v1/tenant/applications/{appID}/menus。
 * 内置应用（platform_admin/tenant_admin）归属平台租户 t_platform，e2e 的 admin 即平台租户管理员。
 *
 * 说明：管理页路由由后端菜单树派生，而 e2e 复用「已初始化」库，不含版本升级新增的内置菜单
 * （tenant-menu），故 /menu 页面路由在旧库上不注册。因此本用例以「真实浏览器登录拿 token →
 * 用同一 token 调后端接口」的方式验证功能（归属应用列表 / 菜单 CRUD / 归属权限），不依赖页面路由，
 * 既真实又不被旧库缺菜单行影响。
 */

// 从浏览器 sessionStorage 取 oidc 用户 access_token（oidc-client-ts 默认 key: oidc.user:<authority>:<client_id>）。
async function extractAccessToken(page: import('@playwright/test').Page): Promise<string> {
  const token = await page.evaluate(() => {
    for (let i = 0; i < sessionStorage.length; i++) {
      const k = sessionStorage.key(i);
      if (k && k.startsWith('oidc.user:')) {
        try {
          const v = JSON.parse(sessionStorage.getItem(k) || '{}');
          if (v.access_token) return v.access_token as string;
        } catch {
          /* ignore */
        }
      }
    }
    return '';
  });
  return token;
}

function apiBase(): string {
  return `${CONFIG.installBaseURL}/v1`;
}

async function apiCall(
  request: APIRequestContext,
  token: string,
  method: 'get' | 'post' | 'put' | 'delete',
  path: string,
  data?: object
) {
  return request[method](`${apiBase()}${path}`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data,
  });
}

test.describe('租户菜单管理', () => {
  test('归属租户可列出归属应用并对其菜单做增删', async ({ page, request }) => {
    // 以租户控制台身份登录（rp1Login 会断言进入租户控制台首页）
    await rp1Login(page);

    const token = await extractAccessToken(page);
    expect(token, '应从 sessionStorage 取到登录 token').toBeTruthy();

    // 1) 归属应用列表（getOwnedApps 口径）：应至少有归属应用（内置 platform_admin/tenant_admin 归属 t_platform）
    const ownedResp = await apiCall(request, token, 'get', '/tenant/applications/owned');
    expect(ownedResp.ok(), `GET /tenant/applications/owned 应 200，实际 ${ownedResp.status()}`).toBeTruthy();
    const owned = await ownedResp.json();
    const ownedList: Array<{ appID: string; code: string; name: string }> = owned?.data?.list ?? owned?.list ?? [];
    expect(ownedList.length, '归属应用列表不应为空（至少含内置应用）').toBeGreaterThan(0);
    const targetApp = ownedList[0];

    // 2) 创建菜单
    const code = `e2e_menu_${Date.now()}`;
    const name = `E2E 菜单 ${Date.now()}`;
    const createResp = await apiCall(request, token, 'post', `/tenant/applications/${targetApp.appID}/menus`, {
      name,
      code,
      type: 'menu',
      visibility: 'member',
      status: 'enable',
    });
    expect(createResp.ok(), `创建菜单应 200，实际 ${createResp.status()}: ${await createResp.text()}`).toBeTruthy();
    const created = await createResp.json();
    const menuID: string = created?.data?.menuID ?? created?.menuID;
    expect(menuID, '创建菜单应返回 menuID').toBeTruthy();

    // 3) 菜单树应包含新菜单
    const treeResp = await apiCall(request, token, 'get', `/tenant/applications/${targetApp.appID}/menus/tree`);
    expect(treeResp.ok()).toBeTruthy();
    const tree = await treeResp.json();
    const treeList: any[] = tree?.data?.list ?? tree?.list ?? [];
    const flat: any[] = [];
    const walk = (nodes: any[]) => nodes.forEach((n) => { flat.push(n); if (n.children) walk(n.children); });
    walk(treeList);
    expect(flat.some((m) => m.menuID === menuID || m.code === code), '菜单树应包含新建菜单').toBeTruthy();

    // 4) 删除菜单
    const delResp = await apiCall(request, token, 'delete', `/tenant/applications/${targetApp.appID}/menus/${menuID}`);
    expect(delResp.ok(), `删除菜单应 200，实际 ${delResp.status()}`).toBeTruthy();

    // 5) 删除后菜单树不再包含
    const treeResp2 = await apiCall(request, token, 'get', `/tenant/applications/${targetApp.appID}/menus/tree`);
    const tree2 = await treeResp2.json();
    const flat2: any[] = [];
    walk0(tree2?.data?.list ?? tree2?.list ?? [], flat2);
    expect(flat2.some((m) => m.menuID === menuID || m.code === code), '删除后菜单树不应再包含该菜单').toBeFalsy();
  });

  test('非归属应用不可维护菜单（归属校验 fail-closed）', async ({ page, request }) => {
    await rp1Login(page);
    const token = await extractAccessToken(page);
    expect(token).toBeTruthy();

    // 用一个随机不存在的 appID：非归属/不存在都应被拒（100606），而非成功。
    // 注意本仓约定：业务错误用 HTTP 200 + body.code 非 0 表达（gincontext.Fail），故断言 body.code。
    const fakeAppID = 'nonexistent-app-' + Date.now();
    const resp = await apiCall(request, token, 'post', `/tenant/applications/${fakeAppID}/menus`, {
      name: 'should-not-create',
      code: 'x',
      type: 'menu',
      visibility: 'member',
      status: 'enable',
    });
    const body = await resp.json();
    expect(body?.code, `非归属/不存在应用建菜单应返回业务错误码（期望 100606），实际 ${JSON.stringify(body)}`).toBe(100606);
  });
});

// walk0：pure 版组树展开（供第二次断言复用）
function walk0(nodes: any[], out: any[]) {
  nodes.forEach((n) => {
    out.push(n);
    if (n.children) walk0(n.children, out);
  });
}
