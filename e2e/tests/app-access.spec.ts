import { test, expect, type APIResponse, type Page } from '@playwright/test';
import { rp1Login } from '../helpers/oidc-helpers';

/**
 * 应用准入（`user_application`）e2e。
 *
 * 运行前提（**必须**先满足，否则用例会 skip 或失败）：
 *  1. 完整活栈已启动（`pnpm -C e2e test` 的 global-setup 会拉起 gateway + 三个前端）；
 *  2. gateway/auth 配置 `oidc.enforceAppAccess: true`（门禁打开）；
 *  3. 环境变量 `IAM_E2E_ENFORCE_APP_ACCESS=1`（显式声明"门禁已打开"，避免在门禁关闭时给出假绿灯）。
 *
 * 为什么用 API 而非 UI 驱动准入读写：`PUT /v1/tenant/users/{userID}/applications` 是准入的唯一写入口，
 * 用例直接打它就是端到端（真 gateway + 真鉴权中间件 + 真库），且不受表格/弹窗选择器漂移影响；
 * 涉及"人看到什么"的部分（无权限页、二次确认文案）由单元/组件测试承担。
 *
 * 登录入口必须是**租户控制台**（`rp1Login` → :4002 / client `tenant_admin_web`）而不是平台控制台：
 * tenantadmin 的 OIDC 中间件按 `model.SeedBuiltinClientTenantAdminWeb` 收紧 `aud`
 * （apps/tenantadmin/app.go:43），平台控制台令牌（aud=platform_admin_web）会被 401 拒绝。
 *
 * ⚠️ 断言口径（本仓统一响应封装）：业务错误**也是 HTTP 200**，只是 `code != 0`
 * （实测："邮箱或手机号至少填写一个" = HTTP 200 + `100521`）。因此一律断言响应体 `code`，
 * **不能**用 `resp.ok()` 判成败——那只是"HTTP 成功"，不等于"业务成功"。
 *
 * 覆盖：AC-3 的准入前置（默认拒绝 → 授权 → 撤销后不可用）、AC-8/I5 防自锁、P6-4 旧路由下线。
 * 未覆盖（由单测承担）：AC-5 静默登录 403 页、AC-1/AC-2 令牌门禁的协议级错误码。
 */
const ENFORCED = process.env.IAM_E2E_ENFORCE_APP_ACCESS === '1';

const API = 'http://localhost:8100/v1/tenant';

interface Envelope<T> {
  code: number;
  msg: string;
  data?: T;
}

/** 解析统一响应封装：默认要求业务成功（code=0），并断言 HTTP 200 这一本仓恒定形态。 */
async function envelope<T>(resp: APIResponse, expectCode = 0): Promise<Envelope<T>> {
  expect(resp.status(), '业务错误与成功都应是 HTTP 200（本仓统一封装）').toBe(200);
  const body = (await resp.json()) as Envelope<T>;
  expect(body.code, `业务码不符，msg=${body.msg}`).toBe(expectCode);
  return body;
}

/** 只取业务码（用于"必须失败"的断言）。 */
async function codeOf(resp: APIResponse): Promise<number> {
  const body = (await resp.json()) as Envelope<unknown>;
  return body.code;
}

/** 从 oidc-client-ts 的 sessionStorage 里取当前用户的 access_token（键形如 oidc.user:<authority>:<client_id>）。 */
async function accessToken(page: Page): Promise<string> {
  const token = await page.evaluate(() => {
    for (let i = 0; i < sessionStorage.length; i += 1) {
      const key = sessionStorage.key(i) || '';
      if (!key.startsWith('oidc.user:')) continue;
      const raw = sessionStorage.getItem(key);
      if (!raw) continue;
      try {
        const parsed = JSON.parse(raw) as { access_token?: string };
        if (parsed.access_token) return parsed.access_token;
      } catch {
        // 非 JSON：忽略该键
      }
    }
    return '';
  });
  expect(token, '未能从 sessionStorage 取到 oidc user 的 access_token').not.toBe('');
  return token;
}

/** 解出 access token 的 payload：用例据此取"本人的 tenant_user.id"，比按 keyword 搜用户更可靠。 */
function jwtClaims(token: string): { user_id?: string; tenant_id?: string; person_id?: string } {
  return JSON.parse(Buffer.from(token.split('.')[1], 'base64url').toString('utf8')) as {
    user_id?: string;
    tenant_id?: string;
    person_id?: string;
  };
}

test.describe('应用准入（user_application）', () => {
  test.skip(!ENFORCED, '需要门禁打开：gateway 配 oidc.enforceAppAccess=true + IAM_E2E_ENFORCE_APP_ACCESS=1');

  test('默认拒绝 → 授权可用 → 撤销后准入关闭（AC-3 前置 + 唯一写入口）', async ({ page, request }) => {
    // 管理员登入租户控制台（RP1 = tenant-admin-web），并从会话里取令牌供 API 调用
    await rp1Login(page);
    const token = await accessToken(page);
    const auth = { Authorization: `Bearer ${token}` };

    // 主部门是建成员的必填入参：取部门树根节点
    const tree = await envelope<{ list: Array<{ departmentID: string }> }>(
      await request.get(`${API}/departments/tree`, { headers: auth }),
    );
    const departmentID = tree.data?.list?.[0]?.departmentID;
    expect(departmentID, '租户应有根部门').toBeTruthy();

    // 本租户订阅的应用清单（原 /v1/tenant/apps，已改名为 /applications）
    const apps = await envelope<{ list: Array<{ appID: string; code: string }> }>(
      await request.get(`${API}/applications`, { headers: auth }),
    );
    const appIDs = (apps.data?.list || []).map((a) => a.appID);
    expect(appIDs.length, '租户应至少订阅一个应用').toBeGreaterThan(0);

    // 造一个"有账号但没有任何准入"的成员（邮箱/手机至少填一个；初始口令仅此一次返回）
    const suffix = Date.now().toString(36);
    const created = await envelope<{ userID: string; initialPassword: string }>(
      await request.post(`${API}/users`, {
        headers: auth,
        data: {
          name: `准入用例成员-${suffix}`,
          username: `appaccess_${suffix}`,
          primaryEmail: `appaccess_${suffix}@example.com`,
          primaryDepartmentID: departmentID,
        },
      }),
    );
    const userID = created.data?.userID;
    expect(userID, '创建成员应返回 userID').toBeTruthy();
    expect(created.data?.initialPassword, '首次创建应一次性返回初始临时口令').toBeTruthy();

    // 默认拒绝：新建成员对任何应用都没有准入
    const before = await envelope<{ list: Array<{ granted: boolean }> }>(
      await request.get(`${API}/users/${userID}/applications`, { headers: auth }),
    );
    expect(before.data?.list?.some((it) => it.granted)).toBe(false);

    // 授权（全量替换）→ 该应用变为已准入，来源 manual
    await envelope(
      await request.put(`${API}/users/${userID}/applications`, { headers: auth, data: { appIDs: [appIDs[0]] } }),
    );
    const grantedList = await envelope<{ list: Array<{ appID: string; granted: boolean; source: string }> }>(
      await request.get(`${API}/users/${userID}/applications`, { headers: auth }),
    );
    const grantedRow = grantedList.data?.list?.find((it) => it.appID === appIDs[0]);
    expect(grantedRow?.granted).toBe(true);
    expect(grantedRow?.source).toBe('manual');

    // 应用视角能查到该成员（撤销前的影响面核对入口）
    const appUsers = await envelope<{ list: Array<{ userID: string }>; total: number }>(
      await request.get(`${API}/applications/${appIDs[0]}/users`, { headers: auth }),
    );
    expect(appUsers.data?.list?.some((u) => u.userID === userID)).toBe(true);

    // 撤销（全量替换为空集）→ 准入关闭，但**行保留**（status=disable，可再授权）
    await envelope(
      await request.put(`${API}/users/${userID}/applications`, { headers: auth, data: { appIDs: [] } }),
    );
    const afterRevoke = await envelope<{ list: Array<{ appID: string; granted: boolean; status: string }> }>(
      await request.get(`${API}/users/${userID}/applications`, { headers: auth }),
    );
    const revokedRow = afterRevoke.data?.list?.find((it) => it.appID === appIDs[0]);
    expect(revokedRow?.granted).toBe(false);
    expect(revokedRow?.status).toBe('disable');

    // 缺字段（而非空数组）必须被拒：防"漏传即清空全部准入"（业务码非 0）
    const missing = await request.put(`${API}/users/${userID}/applications`, { headers: auth, data: {} });
    expect(await codeOf(missing), '缺 appIDs 必须被拒（binding:required）').not.toBe(0);

    // 清理：挂起该成员（准入已撤销，不影响他人）
    await request.patch(`${API}/users/${userID}`, { headers: auth, data: { status: 'suspended' } });
  });

  test('防自锁：不能撤销最后一个内置管理员的控制台准入（AC-8/I5）', async ({ page, request }) => {
    await rp1Login(page);
    const token = await accessToken(page);
    const auth = { Authorization: `Bearer ${token}` };
    const adminUserID = jwtClaims(token).user_id;
    expect(adminUserID, '令牌应带 user_id 声明').toBeTruthy();

    const apps = await envelope<{ list: Array<{ appID: string; code: string }> }>(
      await request.get(`${API}/applications`, { headers: auth }),
    );
    const consoleApp = (apps.data?.list || []).find((a) => a.code === 'tenant_admin');
    expect(consoleApp, '租户控制台应用应已订阅').toBeTruthy();

    // 试图把自己对该控制台的准入撤掉 → 被 I5 拒绝（100787），否则管理员将自我锁死
    const revoke = await request.put(`${API}/users/${adminUserID}/applications`, {
      headers: auth,
      data: { appIDs: [] },
    });
    expect(await codeOf(revoke), '撤销最后一个内置管理员的准入必须被拒').toBe(100787);

    // 更硬的证据：准入必须仍是 enable（I5 是"预检失败零写入"，不是回滚）
    const mine = await envelope<{ list: Array<{ appID: string; granted: boolean }> }>(
      await request.get(`${API}/users/${adminUserID}/applications`, { headers: auth }),
    );
    expect(mine.data?.list?.find((it) => it.appID === consoleApp?.appID)?.granted).toBe(true);
  });

  test('旧路由 /v1/tenant/apps 已下线（P6-4 改名无兼容别名）', async ({ page, request }) => {
    await rp1Login(page);
    const token = await accessToken(page);
    const auth = { Authorization: `Bearer ${token}` };

    const legacy = await request.get(`${API}/apps`, { headers: auth });
    expect(legacy.status(), '旧路径必须已删除，不得保留兼容别名').toBe(404);

    await envelope(await request.get(`${API}/applications`, { headers: auth }));
  });
});
