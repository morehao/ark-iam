import { spawn, execSync, type ChildProcess } from 'child_process';
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
  console.log('[globalSetup] complete\n');

  process.env.E2E_SERVICE_CHILDREN = JSON.stringify(children.map((c) => c.pid));
}

export default globalSetup;
