import { spawn } from 'child_process';

const PORTS = [8100, 4000, 4001, 4002];
const PORT_LABELS: Record<number, string> = {
  8100: 'IAM Backend',
  4000: 'login-web',
  4001: 'platform-admin-web',
  4002: 'tenant-admin-web',
};

function killByPort(port: number, label: string): Promise<void> {
  return new Promise((resolve) => {
    const child = spawn('lsof', ['-ti', `:${port}`], { stdio: ['ignore', 'pipe', 'pipe'] });
    let pidStr = '';
    child.stdout.on('data', (d: Buffer) => { pidStr += d.toString(); });
    child.on('close', () => {
      if (!pidStr.trim()) { resolve(); return; }
      const pids = pidStr.trim().split('\n').map((p) => parseInt(p)).filter(Boolean);
      for (const pid of pids) {
        try { process.kill(pid, 'SIGTERM'); console.log(`  ⏹ ${label} (PID ${pid})`); } catch {}
      }
      // after 2s, force kill remaining
      setTimeout(() => {
        for (const pid of pids) {
          try { process.kill(pid, 0); process.kill(pid, 'SIGKILL'); console.log(`  ☠ ${label} (PID ${pid}) force killed`); } catch {}
        }
        resolve();
      }, 2000);
    });
    child.on('error', () => resolve());
  });
}

// also kill child processes we started
async function killChildren() {
  try {
    const childrenJson = process.env.E2E_SERVICE_CHILDREN;
    if (childrenJson && childrenJson !== '[]') {
      const pids: number[] = JSON.parse(childrenJson);
      for (const pid of pids) {
        try { process.kill(-pid, 'SIGTERM'); } catch {}
      }
    }
  } catch {}
}

async function globalTeardown() {
  console.log('\n[globalTeardown] stopping services...\n');

  // first try to kill via process group
  await killChildren();

  // 再按端口兜底清理——但**只清理 globalSetup 真正启动过的端口**。
  //
  // 曾经这里无条件 PORTS.map(...)：而 globalSetup 对"已经在跑且健康"的服务是**复用**的，
  // 于是开发者在本地开着三个 dev server 跑一次 e2e，收尾会把这些 dev server 全部 SIGTERM/SIGKILL
  // ——setup 说"我不动你"，teardown 却把它们杀了。
  // 现在清理范围来自 setup 上报的 E2E_STARTED_PORTS；读不到（例如 setup 中途崩了）时
  // 保守地什么都不杀：宁可留孤儿进程，也不误杀开发者自己的服务。
  let startedPorts: number[] = [];
  try {
    const raw = process.env.E2E_STARTED_PORTS;
    if (raw) startedPorts = JSON.parse(raw);
  } catch {}
  const reused = PORTS.filter((p) => !startedPorts.includes(p));
  if (reused.length > 0) {
    console.log(`  ℹ️ 保留非本次启动的服务（未按端口清理）：${reused.map((p) => PORT_LABELS[p] ?? p).join('、')}`);
  }
  await Promise.all(startedPorts.map((port) => killByPort(port, PORT_LABELS[port] ?? String(port))));

  console.log('[globalTeardown] complete\n');
}

export default globalTeardown;
