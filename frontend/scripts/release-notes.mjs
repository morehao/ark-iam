#!/usr/bin/env node
/**
 * 从注记 tag 读出发布说明正文（GitHub Release body）。
 *
 * 只做投影、不重新生成：正文里的中文摘要层是发布前确认过的那份，重算必然与已推送内容漂移。
 * 轻量 tag 没有正文（`%(contents)` 会落到 commit message 上），必须拦住，否则会生成一份
 * 看起来像说明、实际是提交信息的 body。
 *
 * 用法：node scripts/release-notes.mjs <tag>（或 RELEASE_TAG=v0.1.1）
 * 输出：body 到 stdout；取不到正文时非零退出，不产出空说明。
 */
import { execFileSync } from 'node:child_process';

const tag = process.argv[2] ?? process.env.RELEASE_TAG;
if (!tag) {
  console.error('用法：release-notes.mjs <tag>（或设置 RELEASE_TAG）');
  process.exit(1);
}

const run = (args) => execFileSync('git', args, { encoding: 'utf8', maxBuffer: 10 * 1024 * 1024 });

let type;
try {
  type = run(['cat-file', '-t', `refs/tags/${tag}`]).trim();
} catch {
  console.error(`✗ 找不到 tag：${tag}（先用 git fetch --tags 取回）`);
  process.exit(1);
}
if (type !== 'tag') {
  console.error(`✗ ${tag} 不是注记 tag（类型 ${type}）：轻量 tag 没有正文可投影`);
  process.exit(1);
}

const raw = run(['for-each-ref', '--format=%(contents)', `refs/tags/${tag}`]);
const lines = raw.replace(/\n+$/, '').split('\n');
lines.shift(); // release: <tag>
if (lines[0] === '') lines.shift();

const body = lines.join('\n');
if (body.trim() === '') {
  console.error(`✗ ${tag} 的注记只有首行，没有变更说明；不生成空 Release`);
  process.exit(1);
}
process.stdout.write(`${body}\n`);
