/**
 * 时间渲染：兼容秒级时间戳与字符串，统一为 `YYYY-MM-DD HH:mm:ss`。
 * 前后端时间交互约定为秒级 int64 时间戳（见 AGENTS.md），前端展示统一走本函数。
 */
export function fmtTime(value?: number | string | null): string {
  if (value == null || value === '' || value === 0) return '-'
  if (typeof value === 'number') {
    const ms = toMillis(value)
    const d = new Date(ms)
    if (Number.isNaN(d.getTime())) return String(value)
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }
  return value
}

/**
 * 时间戳归一化为毫秒：约定入参是秒级时间戳（AGENTS.md），
 * 同时兼容历史上传入的毫秒值（1e12 已远超秒级上限，可安全区分）。
 */
export function toMillis(value: number): number {
  return value < 1e12 ? value * 1000 : value
}

/** 相对时间分档阈值（秒） */
const REL_MINUTE = 60
const REL_HOUR = 60 * REL_MINUTE
const REL_DAY = 24 * REL_HOUR
const REL_MONTH = 30 * REL_DAY
const REL_YEAR = 365 * REL_DAY

/**
 * 相对时间：`刚刚 / N 分钟前 / N 小时前 / N 天前 / N 个月前 / N 年前`。
 *
 * 只用于「最后使用 / 最近使用 / 登录时间」这类"最近"语义的次要时间字段
 * （列窄、扫描时只需粒度感，精确时间由 TimeCell 的 Tooltip 兜底）。
 * - 时间戳晚于当前时刻（时钟漂移等）不猜测为"N 分钟后"，回退绝对时间；
 * - 空值返回空串，由 TimeCell 负责占位文案（如「从未使用」）；
 * - `now` 仅用于测试注入。
 */
export function fmtRelativeTime(value?: number | string | null, now: number = Date.now()): string {
  if (value == null || value === '' || value === 0) return ''
  if (typeof value !== 'number') return String(value)
  const ms = toMillis(value)
  if (Number.isNaN(ms)) return String(value)
  const diff = Math.floor((now - ms) / 1000)
  if (diff < 0) return fmtTime(value)
  if (diff < REL_MINUTE) return '刚刚'
  if (diff < REL_HOUR) return `${Math.floor(diff / REL_MINUTE)} 分钟前`
  if (diff < REL_DAY) return `${Math.floor(diff / REL_HOUR)} 小时前`
  if (diff < REL_MONTH) return `${Math.floor(diff / REL_DAY)} 天前`
  if (diff < REL_YEAR) return `${Math.floor(diff / REL_MONTH)} 个月前`
  return `${Math.floor(diff / REL_YEAR)} 年前`
}
