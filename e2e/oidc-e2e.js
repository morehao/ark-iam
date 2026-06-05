const puppeteer = require('puppeteer-core');

const CONFIG = {
  chromePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  issuer: 'http://localhost:8099/v1/iam/oidc',
  rp1Url: 'http://localhost:3001/',
  rp2Url: 'http://localhost:3002/',
  logWebUrl: 'http://localhost:3003/login',
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

async function clickByText(page, text) {
  return page.evaluate((t) => {
    const btns = [...document.querySelectorAll('button')];
    const target = btns.find((b) => b.textContent.includes(t));
    if (target) {
      target.click();
      return true;
    }
    return false;
  }, text);
}

async function fillFormAndSubmit(page) {
  await page.waitForSelector('#identifier', { timeout: 5000 });
  await page.type('#identifier', CONFIG.identifier);
  await page.type('#password', CONFIG.password);
  await Promise.all([
    page.waitForNavigation({ waitUntil: 'networkidle0', timeout: 15000 }).catch(() => {}),
    page.click('button[type="submit"]'),
  ]);
  await wait(1500);
}

function decodeJwtPayload(tokenStr) {
  const parts = tokenStr.split('.');
  if (parts.length !== 3) return null;
  try {
    const padded = parts[1] + '='.repeat((4 - (parts[1].length % 4)) % 4);
    return JSON.parse(Buffer.from(padded.replace(/-/g, '+').replace(/_/g, '/'), 'base64').toString());
  } catch {
    return null;
  }
}

function extractSubFromJwts(jwts) {
  for (const t of jwts) {
    const payload = decodeJwtPayload(t);
    if (payload?.sub) return payload.sub;
  }
  return null;
}

async function rp1FirstLogin(page) {
  console.log('\n========== Step 5: RP1 首次登录 ==========');

  console.log('1) 打开 http://localhost:3001/');
  await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle0' });
  check('RP1 首页加载', page.url() === CONFIG.rp1Url, page.url());

  console.log('2) 点击"使用 IAM 登录"按钮');
  const clicked = await clickByText(page, '使用 IAM');
  check('找到并点击"使用 IAM 登录"按钮', clicked);
  await wait(2000);

  const logWebUrl = page.url();
  check('跳转到 log-web 登录页',
    logWebUrl.startsWith(CONFIG.logWebUrl) && logWebUrl.includes('authRequestID='),
    logWebUrl);

  console.log('3) 输入 admin/admin123 并提交');
  await fillFormAndSubmit(page);

  const callbackUrl = page.url();
  check('跳回 RP1 回调页 (带 code/state)',
    callbackUrl.startsWith(CONFIG.rp1Url) && callbackUrl.includes('code=') && callbackUrl.includes('state='),
    callbackUrl);

  await page.waitForFunction(
    () => document.body.innerText.includes('项目管理面板'),
    { timeout: 10000 }
  );
  const homeText = await page.evaluate(() => document.body.innerText);
  check('显示"项目管理面板"标题', homeText.includes('项目管理面板'));
  check('显示 SSO 登录徽标', homeText.includes('已通过 IAM SSO 登录'));

  const statCardCount = await page.$$eval('.stat-card', (els) => els.length);
  check('展示 4 个统计卡片', statCardCount === 4, `实际 ${statCardCount} 个`);
  check('统计卡片包含"项目数"', homeText.includes('项目数'));
  check('统计卡片包含"任务数"', homeText.includes('任务数'));
  check('统计卡片包含"消息数"', homeText.includes('消息数'));
  check('统计卡片包含"团队数"', homeText.includes('团队数'));
}

async function testTokenDetails(page) {
  console.log('\n========== 验证 Token 详情相关功能 ==========');

  await clickByText(page, '查看 Token 详情');
  await wait(1000);

  const tokenText = await page.evaluate(() => document.body.innerText);
  check('Token 详情页显示 access_token', tokenText.includes('access_token'));
  check('Token 详情页显示 id_token', tokenText.includes('id_token'));
  check('Token 详情页显示 refresh_token', tokenText.includes('refresh_token'));

  const idTokenJwts = await page.evaluate(() => {
    const text = document.body.innerText;
    return text.match(/eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/g) || [];
  });
  const sub = extractSubFromJwts(idTokenJwts);
  check('从 id_token 解码出 sub (personID)', !!sub, sub || '未找到');

  console.log('点击"获取 UserInfo"');
  await clickByText(page, '获取 UserInfo');
  await wait(2000);
  const userinfoText = await page.evaluate(() => document.body.innerText);
  check('UserInfo 获取成功',
    ['"name"', '"username"', '"email"', '"sub"'].some((k) => userinfoText.includes(k)));

  console.log('点击"刷新 Token"');
  const tokensBefore = await page.evaluate(() => currentTokens?.access_token);
  await clickByText(page, '刷新 Token');
  await wait(3000);
  const tokensAfter = await page.evaluate(() => currentTokens?.access_token);
  check('Token 刷新成功 (access_token 已更新)',
    !!tokensBefore && !!tokensAfter && tokensBefore !== tokensAfter,
    `before=len${tokensBefore?.length} after=len${tokensAfter?.length}`);

  console.log('点击"返回主页"');
  await clickByText(page, '返回主页');
  await wait(1000);
  const homeText = await page.evaluate(() => document.body.innerText);
  check('成功返回项目管理面板', homeText.includes('项目管理面板'));

  return sub;
}

async function ssoTest(rp1Page, rp1Sub) {
  console.log('\n========== Step 6: 双 RP SSO 验证 ==========');
  const browser = rp1Page.browser();
  const page2 = await browser.newPage();

  const cookies = await rp1Page.cookies();
  await page2.setCookie(...cookies);
  console.log(`已共享 ${cookies.length} 个 cookies`);

  const ssoCookie = cookies.find((c) => c.name === 'iam_sso_session');
  check('iam_sso_session cookie 存在', !!ssoCookie,
    ssoCookie ? `domain=${ssoCookie.domain}, value=${ssoCookie.value.substring(0, 8)}...` : '无');

  console.log('2) 打开 http://localhost:3002/');
  await page2.goto(CONFIG.rp2Url, { waitUntil: 'networkidle0', timeout: 15000 });
  await wait(2000);

  const loginFormVisible = await page2.$('form input[type="text"]');
  check('SSO 流程中无登录表单出现', !loginFormVisible);

  try {
    await page2.waitForFunction(
      () => document.body.innerText.includes('数据分析面板'),
      { timeout: 10000 }
    );
  } catch {
    // 容忍 timeout，下面用文本判断
  }

  const rp2FinalText = await page2.evaluate(() => document.body.innerText);
  check('RP2 显示"数据分析面板"', rp2FinalText.includes('数据分析面板'));
  check('RP2 显示 SSO 登录徽标',
    rp2FinalText.includes('SSO 自动登录') || rp2FinalText.includes('已通过 IAM SSO 登录'));
  check('RP2 展示 4 个统计卡片 (订单数/客户数/待处理/完成率)',
    ['订单数', '客户数', '待处理', '完成率'].every((k) => rp2FinalText.includes(k)));

  await clickByText(page2, '查看 Token 详情');
  await wait(2000);
  const rp2Jwts = await page2.evaluate(() => {
    const text = document.body.innerText;
    return text.match(/eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/g) || [];
  });
  const rp2IdTokenSub = extractSubFromJwts(rp2Jwts);
  check('RP2 id_token 解码出 sub', !!rp2IdTokenSub, rp2IdTokenSub || '未找到');
  check('RP1 和 RP2 的 id_token.sub 一致',
    rp1Sub && rp2IdTokenSub && rp1Sub === rp2IdTokenSub,
    `RP1=${rp1Sub} RP2=${rp2IdTokenSub}`);

  await page2.close();
}

(async () => {
  const browser = await puppeteer.launch({
    executablePath: CONFIG.chromePath,
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--incognito'],
  });

  const page = await browser.newPage();
  page.setDefaultTimeout(30000);

  try {
    await rp1FirstLogin(page);
    const rp1Sub = await testTokenDetails(page);
    await ssoTest(page, rp1Sub);
  } catch (e) {
    console.error('\n[!] 测试异常:', e.message);
    console.error(e.stack);
    fail++;
  } finally {
    console.log('\n========== 测试结果汇总 ==========');
    results.forEach((r) => console.log(r));
    console.log(`\n总计: ${pass + fail} 通过: ${pass} 失败: ${fail}`);
    await browser.close();
    process.exit(fail > 0 ? 1 : 0);
  }
})();
