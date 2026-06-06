import { spawn } from 'child_process';

const PORTS = [8099, 3000, 3001, 3003];
const PORT_LABELS: Record<number, string> = {
  8099: 'IAM Backend',
  3000: 'platform-admin-web',
  3001: 'sso-test-app',
  3003: 'login-web',
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

  // then clean up by port
  await Promise.all(PORTS.map((port) => killByPort(port, PORT_LABELS[port])));

  console.log('[globalTeardown] complete\n');
}

export default globalTeardown;
