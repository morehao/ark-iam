import { spawn, execSync, type ChildProcess } from 'child_process';
import { CONFIG } from './config';
import * as http from 'http';
import * as net from 'net';
import * as path from 'path';

const ROOT = path.resolve(__dirname, '..');
const BACKEND_ROOT = path.join(ROOT, 'backend');
const FRONTEND_ROOT = path.join(ROOT, 'frontend');

interface ServiceDef {
  name: string;
  port: number;
  cmd: string;
  args: string[];
  cwd: string;
  env?: Record<string, string>;
  /** 健康检查路径，用于验证已运行的服务是否可用 */
  healthPath: string;
}

const SERVICES: ServiceDef[] = [
  {
    name: 'IAM Backend',
    port: 8100,
    // 优先使用预编译二进制（backend/.tmp/gateway-bin），避免 go run 首次编译
    // 缓慢导致健康检查超时；无二进制时回退 go run。
    cmd: 'go',
    args: ['run', './apps/gateway/cmd'],
    cwd: path.join(ROOT, 'backend'),
    env: {
      APP_CONFIG_PATH: path.join(ROOT, 'backend', 'apps', 'gateway', 'config', 'config.yaml'),
      // 初始化引导的一次性令牌。启动期播种已删除，因此**每次全新库跑 e2e 都必须有人**
      // 调用 POST /install/initialize——本文件末尾的 ensureInitialized 就是那个调用方。
      // 未配置该变量时 /install/initialize 整体不可用（fail-closed），e2e 会直接失败并给出提示。
      BOOTSTRAP_TOKEN: CONFIG.bootstrapToken,
    },
    healthPath: '/oidc/healthz',
  },
  {
    name: 'login-web',
    port: 3000,
    cmd: 'pnpm',
    args: ['--filter', '@ark-iam/login-web', 'dev'],
    cwd: FRONTEND_ROOT,
    healthPath: '/',
  },
  {
    name: 'platform-admin-web',
    port: 3001,
    cmd: 'pnpm',
    args: ['--filter', '@ark-iam/platform-admin-web', 'dev'],
    cwd: FRONTEND_ROOT,
    healthPath: '/',
  },
  {
    name: 'tenant-admin-web',
    port: 3002,
    cmd: 'pnpm',
    args: ['--filter', '@ark-iam/tenant-admin-web', 'dev'],
    cwd: FRONTEND_ROOT,
    healthPath: '/',
  },
];

// 若预编译 gateway 二进制存在，则替换 backend 服务定义，避免 go run 编译慢导致超时。
const GATEWAY_BIN = path.join(BACKEND_ROOT, '.tmp', 'gateway-bin');
const fs = require('fs') as typeof import('fs');
if (fs.existsSync(GATEWAY_BIN)) {
  SERVICES[0] = {
    ...SERVICES[0],
    cmd: GATEWAY_BIN,
    args: [],
  };
}

async function checkPort(port: number): Promise<boolean> {
  return new Promise((resolve) => {
    const hosts = ['127.0.0.1', '::1'];
    let tried = 0;
    for (const host of hosts) {
      const socket = new net.Socket();
      socket.setTimeout(1500);
      socket.once('connect', () => { socket.destroy(); resolve(true); });
      socket.once('error', () => { socket.destroy(); });
      socket.once('timeout', () => { socket.destroy(); });
      socket.once('close', () => { tried++; if (tried === hosts.length) resolve(false); });
      socket.connect(port, host);
    }
  });
}

/**
 * 对已占用端口的服务做健康检查，确保它能正常响应。
 * 返回 true 表示服务健康可用，false 表示需要重启。
 */
function healthCheck(port: number, healthPath: string, timeoutMs: number = 5000): Promise<boolean> {
  return new Promise((resolve) => {
    const req = http.get(`http://127.0.0.1:${port}${healthPath}`, { timeout: timeoutMs }, (res) => {
      // 2xx/3xx 认为健康
      const status = res.statusCode ?? 0;
      resolve(status >= 200 && status < 400);
    });
    req.on('error', () => resolve(false));
    req.on('timeout', () => { req.destroy(); resolve(false); });
  });
}

/**
 * 强行杀掉指定端口上的所有进程，确保端口干净可用。
 */
function killPort(port: number, label: string): void {
  try {
    const pids = execSync(`lsof -ti:${port}`, { encoding: 'utf-8' })
      .trim()
      .split('\n')
      .filter(Boolean);
    for (const pid of pids) {
      try {
        process.kill(parseInt(pid), 'SIGKILL');
        console.log(`  ⏹ killed stale ${label} (PID ${pid}) on port ${port}`);
      } catch {}
    }
  } catch {}
}

async function waitForPort(port: number, label: string, timeoutMs: number): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (await checkPort(port)) return true;
    await new Promise((r) => setTimeout(r, 1000));
  }
  return false;
}

const children: ChildProcess[] = [];

async function globalSetup() {
  console.log('\n[globalSetup] checking and starting services...\n');

  const needStart: ServiceDef[] = [];

  for (const svc of SERVICES) {
    const running = await checkPort(svc.port);
    if (running) {
      const healthy = await healthCheck(svc.port, svc.healthPath);
      if (healthy) {
        console.log(`  ✅ ${svc.name} (port ${svc.port}) already running & healthy`);
      } else {
        console.log(`  ⚠️ ${svc.name} (port ${svc.port}) occupied but unhealthy, restarting...`);
        killPort(svc.port, svc.name);
        needStart.push(svc);
      }
    } else {
      console.log(`  ⏳ ${svc.name} (port ${svc.port}) starting...`);
      needStart.push(svc);
    }
  }

  if (needStart.length === 0) {
    console.log('  All services ready\n');
    process.env.E2E_SERVICE_CHILDREN = JSON.stringify([]);
    return;
  }

  await Promise.all(
    needStart.map(
      (svc) =>
        new Promise<void>((resolve, reject) => {
          const opts: any = {
            cwd: svc.cwd,
            stdio: ['ignore', 'pipe', 'pipe'],
            detached: true,
          };
          if (svc.env) {
            opts.env = { ...process.env, ...svc.env };
          }
          const child = spawn(svc.cmd, svc.args, opts);
          children.push(child);

          child.stdout?.on('data', () => {});
          child.stderr?.on('data', () => {});

          child.on('error', (err) => {
            console.error(`[${svc.name}] spawn error: ${err.message}`);
            reject(err);
          });

          waitForPort(svc.port, svc.name, 180000).then((ok) => {
            if (ok) {
              console.log(`  ✅ ${svc.name} ready`);
              resolve();
            } else {
              console.error(`  ❌ ${svc.name} timed out`);
              reject(new Error(`${svc.name} startup timeout`));
            }
          });
        })
    )
  );

  console.log('  All services ready\n');

  await ensureInitialized();
  await verifyInitialized();

  console.log('[globalSetup] complete\n');

  process.env.E2E_SERVICE_CHILDREN = JSON.stringify(children.map((c) => c.pid));
}

/** 极简 JSON 请求（不引 playwright 的 request，globalSetup 运行在测试框架之外）。 */
function jsonRequest(
  method: string,
  url: string,
  body?: unknown,
  headers: Record<string, string> = {}
): Promise<{ status: number; body: any }> {
  return new Promise((resolve, reject) => {
    const payload = body === undefined ? undefined : JSON.stringify(body);
    const req = http.request(
      url,
      {
        method,
        timeout: 30000,
        headers: {
          ...(payload ? { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(payload) } : {}),
          ...headers,
        },
      },
      (res) => {
        let data = '';
        res.on('data', (c) => (data += c));
        res.on('end', () => {
          let parsed: any = null;
          try {
            parsed = data ? JSON.parse(data) : null;
          } catch {
            parsed = data;
          }
          resolve({ status: res.statusCode ?? 0, body: parsed });
        });
      }
    );
    req.on('error', reject);
    req.on('timeout', () => { req.destroy(new Error('request timeout')); });
    if (payload) req.write(payload);
    req.end();
  });
}

/**
 * 确保系统已完成首次初始化。
 *
 * 为什么必须由 e2e 自己做这件事：启动期播种已删除（这是本设计的核心），
 * 后端起来只是一个**空库**——没有平台租户、没有管理员，登录页对谁都登不进去。
 * 若这里不引导，后续所有用例都会以"登录失败"告终，而根因与本用例要验证的东西毫无关系。
 *
 * 走的是与初始化页面**完全相同**的接口（D8：不引入第二条播种路径）。
 * 库已初始化时直接跳过：自锁是产品语义，e2e 不做"重置后再初始化"（那需要直接改库）。
 */
async function ensureInitialized(): Promise<void> {
  const base = CONFIG.installBaseURL;
  const status = await jsonRequest('GET', `${base}/install/status`);
  if (status.status !== 200) {
    throw new Error(
      `[globalSetup] GET /install/status 返回 ${status.status}；后端可能未就绪或未建表（检查 db.auto_migrate）`
    );
  }
  const data = status.body?.data ?? {};
  if (data.tokenRequired === false) {
    throw new Error(
      '[globalSetup] 后端未配置 BOOTSTRAP_TOKEN，/install/initialize 不可用（fail-closed）。' +
        ' e2e 在 SERVICES 的 IAM Backend env 中注入该变量；若后端已在运行，请先停掉再跑 e2e。'
    );
  }
  if (data.initialized === true) {
    console.log('  ✅ 系统已初始化，跳过首次引导');
    return;
  }
  if (data.schemaReady === false) {
    throw new Error('[globalSetup] 表结构未就绪（schemaReady=false）：确认 db.auto_migrate 为 true');
  }

  const res = await jsonRequest(
    'POST',
    `${base}/install/initialize`,
    {
      tenantName: 'E2E 平台运营中心',
      adminUsername: CONFIG.identifier,
      adminPassword: CONFIG.password,
      adminEmail: 'admin@example.com',
      adminName: '系统管理员',
    },
    { 'X-Bootstrap-Token': CONFIG.bootstrapToken }
  );
  if (res.status !== 200) {
    throw new Error(
      `[globalSetup] 首次引导失败（HTTP ${res.status}, code=${res.body?.code}）：${res.body?.msg ?? ''}`
    );
  }
  const created = res.body?.data?.report?.changes?.length ?? 0;
  console.log(`  ✅ 首次引导完成（创建 ${created} 条内置数据，管理员 ${CONFIG.identifier}）`);
}

/**
 * 引导后自检：状态必须翻转为 initialized。
 *
 * 这条断言把"引导其实没生效"与"后续用例登录失败"区分开——
 * 否则一个失败的初始化会表现成十几个互不相关的登录用例失败。
 */
async function verifyInitialized(): Promise<void> {
  const started = Date.now();
  while (Date.now() - started < 10000) {
    const st = await jsonRequest('GET', `${CONFIG.installBaseURL}/install/status`);
    if (st.body?.data?.initialized === true) return;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error('[globalSetup] 引导后 /install/status 仍报未初始化');
}

export default globalSetup;
