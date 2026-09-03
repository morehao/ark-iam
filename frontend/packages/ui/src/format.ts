/**
 * 时间渲染：兼容秒级时间戳与字符串，统一为 `YYYY-MM-DD HH:mm:ss`。
 * 前后端时间交互约定为秒级 int64 时间戳（见 AGENTS.md），前端展示统一走本函数。
 */
export function fmtTime(value?: number | string | null): string {
  if (value == null || value === '' || value === 0) return '-'
  if (typeof value === 'number') {
    const ms = value < 1e12 ? value * 1000 : value
    const d = new Date(ms)
    if (Number.isNaN(d.getTime())) return String(value)
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }
  return value
}
