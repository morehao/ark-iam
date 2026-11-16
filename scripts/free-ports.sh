#!/usr/bin/env bash
# 释放开发端口：结束在指定 TCP 端口上 LISTEN 的进程（SIGTERM → 超时 SIGKILL）。
#
# 解决问题：上次 `make dev-all` 的 gateway / vite 进程残留（终端被关、进程被 SIGKILL、
# 或手工 `go run` 过）后，再启动会报
#   listen tcp :8100: bind: address already in use   → gateway panic
# 或 vite 发现端口被占后静默 +1（:4001 → :4003），导致 OIDC redirect_uri / 代理配置对不上。
#
# 用法：
#   bash scripts/free-ports.sh 8100 4000 4001 4002   # 释放这些端口
#   NO_KILL=1 bash scripts/free-ports.sh 8100        # 只探测，不杀进程
#
# 退出码：0=全部端口已空闲（或已成功释放）；1=有端口释放失败/仍被占用；2=参数错误。
# 说明：只处理 LISTEN 状态的套接字，不碰 TIME_WAIT / 已建立连接的会话。
set -uo pipefail

TERM_TIMEOUT_SEC=$(( ${FREE_PORTS_TERM_TIMEOUT:-5} ))
NO_KILL="${NO_KILL:-0}"

# 探测工具：优先 lsof（macOS / Linux 常见），退化到 ss、fuser。
pick_port_probe() {
  if command -v lsof >/dev/null 2>&1; then
    echo lsof
  elif command -v ss >/dev/null 2>&1; then
    echo ss
  elif command -v fuser >/dev/null 2>&1; then
    echo fuser
  else
    return 1
  fi
}

PORT_PROBE="$(pick_port_probe)" || {
  echo "❌ 未找到 lsof / ss / fuser，无法检测端口占用；请安装其中之一（如 lsof）后重试" >&2
  exit 1
}

# 打印端口上 LISTEN 的 PID（每行一个，去重）。无占用时输出为空且返回 0。
port_pids() {
  local port="$1"
  case "$PORT_PROBE" in
    lsof)
      lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null | sort -u
      ;;
    ss)
      ss -ltnpH "sport = :$port" 2>/dev/null | sed -n 's/.*pid=\([0-9][0-9]*\).*/\1/p' | sort -u
      ;;
    fuser)
      fuser -n tcp "$port" 2>/dev/null | tr ' ' '\n' | grep -E '^[0-9]+$' | sort -u
      ;;
  esac || true
  return 0
}

# 进程命令行（取不到时返回空串，不影响结束进程）。
pid_command() {
  local pid="$1" cmd
  cmd="$(ps -p "$pid" -o command= 2>/dev/null | sed 's/^[[:space:]]*//' | cut -c1-120 || true)"
  if [ -z "$cmd" ] && command -v lsof >/dev/null 2>&1; then
    # ps 不可用时退化为取可执行文件路径（容器 / 受限环境）
    cmd="$(lsof -p "$pid" -a -d txt -Fn 2>/dev/null | sed -n 's/^n//p' | head -1 | cut -c1-120 || true)"
  fi
  printf '%s' "$cmd"
}

# 结束端口上的监听进程；返回 0 表示该端口最终空闲。
free_port() {
  local port="$1" pids pid alive i cmd
  pids="$(port_pids "$port")"

  if [ -z "$pids" ]; then
    echo "✅ 端口 :$port 空闲"
    return 0
  fi

  for pid in $pids; do
    cmd="$(pid_command "$pid")"
    if [ -n "$cmd" ]; then
      echo "⚠️  端口 :$port 被占用 → PID ${pid}（${cmd}），准备结束"
    else
      echo "⚠️  端口 :$port 被占用 → PID ${pid}，准备结束"
    fi
  done

  if [ "$NO_KILL" = "1" ]; then
    echo "ℹ️  NO_KILL=1，仅探测不结束进程"
    return 0
  fi

  kill $pids 2>/dev/null || true

  # 等进程退出（0.1s 粒度轮询，最多 TERM_TIMEOUT_SEC 秒）
  i=0
  while [ "$i" -lt "$(( TERM_TIMEOUT_SEC * 10 ))" ]; do
    alive=""
    for pid in $pids; do
      if kill -0 "$pid" 2>/dev/null; then
        alive="$alive $pid"
      fi
    done
    [ -z "$alive" ] && break
    sleep 0.1
    i=$(( i + 1 ))
  done

  if [ -n "${alive:-}" ]; then
    echo "⚠️  PID$alive 未在 ${TERM_TIMEOUT_SEC}s 内退出，强制结束（SIGKILL）"
    kill -9 $alive 2>/dev/null || true
    sleep 0.2
  fi

  if [ -n "$(port_pids "$port")" ]; then
    echo "❌ 端口 :$port 仍被占用（可能属于其它用户的进程，需手动处理）" >&2
    return 1
  fi

  echo "✅ 端口 :$port 已释放"
  return 0
}

if [ "$#" -eq 0 ]; then
  echo "用法: $0 <port> [port...]" >&2
  exit 2
fi

rc=0
for port in "$@"; do
  case "$port" in
    ''|*[!0-9]*)
      echo "❌ 非法端口：$port" >&2
      rc=1
      continue
      ;;
  esac
  free_port "$port" || rc=1
done

exit "$rc"
