import { spawn, type ChildProcess } from 'child_process';
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
}

const SERVICES: ServiceDef[] = [
  {
    name: 'IAM Backend',
    port: 8099,
    cmd: 'go',
    args: ['run', './apps/iam/cmd'],
    cwd: path.join(ROOT, 'backend'),
    env: {
      APP_CONFIG_PATH: path.join(ROOT, 'backend', 'apps', 'iam', 'config', 'config.yaml'),
    },
  },
  {
    name: 'platform-admin-web',
    port: 3000,
    cmd: 'pnpm',
    args: ['--filter', '@ark-iam/platform-admin-web', 'dev'],
    cwd: FRONTEND_ROOT,
  },
  {
    name: 'sso-test-app',
    port: 3001,
    cmd: 'pnpm',
    args: ['--filter', '@ark-iam/sso-test-app', 'dev'],
    cwd: FRONTEND_ROOT,
  },
  {
    name: 'login-web',
    port: 3003,
    cmd: 'pnpm',
    args: ['--filter', '@ark-iam/login-web', 'dev'],
    cwd: FRONTEND_ROOT,
  },
];

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
      console.log(`  ✅ ${svc.name} (port ${svc.port}) already running`);
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
