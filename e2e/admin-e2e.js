const puppeteer = require('puppeteer-core');

const CONFIG = {
  chromePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  adminUrl: 'http://localhost:3000/',
  identifier: 'admin',
  password: 'admin123',
};

let pass = 0;
let fail = 0;
const results = [];

function check(name, condition, detail = '') {
  if (condition) {
    pass++;
    results.push(`✅ ${name}`);
    console.log(`✅ ${name}`);
  } else {
    fail++;
    const suffix = detail ? ` (${detail})` : '';
    results.push(`❌ ${name}${suffix}`);
    console.log(`❌ ${name}${suffix}`);
  }
}

const wait = (ms) => new Promise((r) => setTimeout(r, ms));

(async () => {
  const browser = await puppeteer.launch({
    executablePath: CONFIG.chromePath,
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const page = await browser.newPage();
  page.setDefaultTimeout(30000);

  // 拦截网络请求，记录 API 调用
  const apiCalls = [];
  page.on('request', (req) => {
    const url = req.url();
    if (url.includes('/v1/iam/')) {
      apiCalls.push({ url, method: req.method(), status: null });
    }
  });
  page.on('response', (resp) => {
    const url = resp.url();
    if (url.includes('/v1/iam/')) {
      const existing = apiCalls.filter((c) => c.url === url && !c.status);
      for (const c of existing) {
        c.status = resp.status();
      }
    }
  });

  try {
    // ============================================================
    // Step 1: 打开管理平台，验证登录页
    // ============================================================
    console.log('\n========== Step 1: 打开管理平台登录页 ==========');

    await page.goto(CONFIG.adminUrl, { waitUntil: 'networkidle0', timeout: 15000 });
    await wait(1000);

    const loginText = await page.evaluate(() => document.body.innerText);
    check('显示"IAM 管理平台"标题', loginText.includes('IAM 管理平台'));
    check('显示"IAM 账号登录"按钮', loginText.includes('IAM 账号登录'));
    check('当前在 /login 路由', page.url().includes('/login'), page.url());

    // ============================================================
    // Step 2: 点击"IAM 账号登录"按钮，触发 OIDC 流程
    // ============================================================
    console.log('\n========== Step 2: OIDC 授权码流程登录 ==========');

    // 点击"IAM 账号登录"按钮，触发 PKCE + 重定向
    await Promise.all([
      page.waitForNavigation({ waitUntil: 'networkidle0', timeout: 15000 }),
      page.click('button'),
    ]);
    await wait(1500);

    const afterClickUrl = page.url();
    console.log(`OIDC 登录后 URL: ${afterClickUrl}`);

    // 检查是否被重定向到 log-web 登录页 (localhost:3003)
    check('重定向到 OIDC 登录页',
      afterClickUrl.includes('localhost:3003') && afterClickUrl.includes('authRequestID='),
      afterClickUrl);

    // ============================================================
    // Step 3: 在 log-web 登录页输入凭证并提交
    // ============================================================
    console.log('\n========== Step 3: 输入凭证 ==========');

    await page.waitForSelector('#identifier', { timeout: 5000 });
    await page.type('#identifier', CONFIG.identifier);
    await page.type('#password', CONFIG.password);

    check('输入框 #identifier 存在', true);
    check('输入框 #password 存在', true);

    // 点击"登录"提交按钮
    const submitClicked = await page.evaluate(() => {
      const btns = document.querySelectorAll('button');
      const loginBtn = Array.from(btns).find((b) => b.textContent?.trim() === '登录');
      if (loginBtn) {
        loginBtn.click();
        return true;
      }
      // fallback: type=submit
      const submitBtn = document.querySelector('button[type="submit"]');
      if (submitBtn) {
        submitBtn.click();
        return true;
      }
      return false;
    });
    check('点击"登录"按钮', submitClicked);

    // 等待 OIDC 回调完成，回到管理平台
    // 流程: POST /v1/iam/oidc/login -> 重定向到 IAM 后端继续 OIDC -> 重定向到 /auth/callback -> 重定向到 /
    await wait(3000);

    // 可能经过多次跳转，等待最终到达仪表盘
    try {
      await page.waitForFunction(
        () => {
          const text = document.body.innerText;
          return text.includes('仪表盘') || text.includes('用户管理');
        },
        { timeout: 15000 }
      );
    } catch {
      // 如果还在回调页面，再等一会
    }

    const afterLoginUrl = page.url();
    console.log(`登录后 URL: ${afterLoginUrl}`);

    // ============================================================
    // Step 4: 验证登录成功 - 仪表盘页面
    // ============================================================
    console.log('\n========== Step 4: 验证登录后仪表盘 ==========');

    const dashboardText = await page.evaluate(() => document.body.innerText);
    check('登录成功 - 显示"仪表盘"', dashboardText.includes('仪表盘'));
    check('显示侧边栏"IAM 管理平台"', dashboardText.includes('IAM 管理平台'));
    check('显示"用户管理"菜单', dashboardText.includes('用户管理'));
    check('显示"角色管理"菜单', dashboardText.includes('角色管理'));
    check('显示"部门管理"菜单', dashboardText.includes('部门管理'));
    check('显示"应用管理"菜单', dashboardText.includes('应用管理'));

    // 检查右上角是否有用户头像（表示用户信息已加载）
    const avatarEl = await page.$('.ant-avatar');
    check('用户头像已加载（getUserinfo API 调用成功）', !!avatarEl);

    // ============================================================
    // Step 5: 验证首页 API 调用
    // ============================================================
    console.log('\n========== Step 5: 验证首页相关 API 调用 ==========');

    console.log('拦截到的 /v1/iam/ API 调用:');
    apiCalls.forEach((c) => console.log(`  ${c.method} ${c.url} => ${c.status || 'pending'}`));

    // 验证 userinfo API 调用
    const userinfoCall = apiCalls.find((c) => c.url.includes('/auth/userinfo'));
    check('首页加载时调用了 GET /auth/userinfo',
      !!userinfoCall && userinfoCall.status === 200,
      userinfoCall ? `status=${userinfoCall.status}` : '未找到该 API 调用');

    // 验证 token 交换 API 调用 (POST /oauth/token)
    const tokenCall = apiCalls.find((c) => c.url.includes('/oauth/token'));
    check('OIDC token 交换成功 (POST /oauth/token)',
      !!tokenCall && tokenCall.status === 200,
      tokenCall ? `status=${tokenCall.status}` : '未找到');

    // 验证 localStorage 中有 auth-storage（token 已持久化）
    const authStorage = await page.evaluate(() => {
      try {
        const raw = localStorage.getItem('auth-storage');
        return raw ? JSON.parse(raw) : null;
      } catch {
        return null;
      }
    });
    check('localStorage 中有 auth-storage（token 已持久化）',
      !!authStorage?.state?.accessToken,
      authStorage?.state?.accessToken
        ? `accessToken: ${authStorage.state.accessToken.substring(0, 20)}...`
        : '无 accessToken');

    // ============================================================
    // Step 6: 测试登出
    // ============================================================
    console.log('\n========== Step 6: 测试登出 ==========');

    // 点击右上角头像打开下拉菜单（使用 Puppeteer 原生 click 确保触发 React 事件）
    await page.click('.ant-avatar');
    await page.waitForSelector('.ant-dropdown-menu-item', { visible: true, timeout: 5000 });
    await wait(500);

    // 点击"退出登录"
    const logoutClicked = await page.evaluate(() => {
      const items = document.querySelectorAll('.ant-dropdown-menu-item');
      for (const item of items) {
        if (item.textContent.includes('退出登录')) {
          item.click();
          return true;
        }
      }
      return false;
    });
    check('找到并点击"退出登录"菜单项', logoutClicked);

    if (logoutClicked) {
      // 登出触发 end_session 请求，等待后端处理并重定向
      await wait(5000);
      const afterLogoutUrl = page.url();
      console.log(`登出后 URL: ${afterLogoutUrl}`);

      // end_session 端点应重定向到 post_logout_redirect_uri（即 /login）
      const isLogoutSuccess =
        afterLogoutUrl.includes('/login') ||
        afterLogoutUrl.includes('end_session'); // end_session 本身也说明登出流程已触发
      check('登出流程已触发（end_session 被调用）',
        isLogoutSuccess,
        `URL: ${afterLogoutUrl}`);

      // 手动导航到 /login 验证 token 已清除
      await page.goto(CONFIG.adminUrl, { waitUntil: 'networkidle0', timeout: 10000 }).catch(() => {});
      await wait(1000);

      const afterLoginPageText = await page.evaluate(() => document.body.innerText);
      check('登出后访问管理平台显示登录页',
        afterLoginPageText.includes('IAM 账号登录'),
        `页面内容: ${afterLoginPageText.substring(0, 50)}`);

      const authAfterLogout = await page.evaluate(() => {
        const raw = localStorage.getItem('auth-storage');
        if (!raw) return null;
        try { return JSON.parse(raw); } catch { return null; }
      });
      check('登出后 accessToken 已清除',
        !authAfterLogout?.state?.accessToken,
        authAfterLogout?.state?.accessToken ? '仍存在' : '已清除');
    }

  } catch (e) {
    console.error('\n[!] 测试异常:', e.message);
    console.error(e.stack);
    fail++;
  } finally {
    console.log('\n========== 管理平台 E2E 测试结果汇总 ==========');
    results.forEach((r) => console.log(r));
    console.log(`\n总计: ${pass + fail} 通过: ${pass} 失败: ${fail}`);

    await browser.close();
    process.exit(fail > 0 ? 1 : 0);
  }
})();
