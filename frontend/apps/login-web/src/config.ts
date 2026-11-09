/**
 * 运行时基址解析（内嵌形态自适应挂载前缀）。
 *
 * 背景：login-web 既可能被**根路径挂载**（gateway / 独立 auth：`/oidc`、`/install` 在 origin 根），
 * 也可能被**非空前缀内嵌**（如 `/iam`：`/iam/oidc`、`/iam/install`）。页面路由也随前缀走
 * （`{base}/login`、`{base}/install`）。因此不能在构建期把 base 写死（`vite base` 只能定一个值，
 * 无法同 dist 多前缀复用），改为**运行时**：页面先探测自身 basePath，拉取 `{basePath}/config.json`
 * 拿到后端下发的带前缀 base，再据此配置 axios 与路由。
 *
 * config.json 由后端 `MountFrontend` 下发（`{}`，字段见 RuntimeConfig），与登录页同源，
 * 内容仅公开配置（issuer / 各 base / callbackPath / clientId），不含密钥。
 */

/** 后端 `GET {basePath}/config.json` 下发的运行时配置。 */
export interface RuntimeConfig {
  /** OIDC issuer 绝对地址（如 http://host/iam/oidc）。 */
  issuer: string
  /** 业务 API 基址（如 /iam/v1）。 */
  apiBase: string
  /** OIDC 端点基址（如 /iam/oidc）。 */
  oidcBase: string
  /** install 端点基址（如 /iam/install）。 */
  installBase: string
  /** 登录页在本挂载下的路径（如 /iam/login）。 */
  loginPath: string
  /** 前端回调路径（相对，如 /auth/callback）。 */
  callbackPath: string
  /** 控制台 OIDC client_id。 */
  clientId: string
}

/**
 * 从当前页面 URL 推断 IAM 挂载 basePath。
 *
 * 页面路由恒为 `{basePath}/login` 或 `{basePath}/install`（含尾随子路径），据此剥离得到
 * basePath：根挂载时返回 ""（如 `/login` → ""），`/iam/login` → "/iam"。
 */
export function detectBasePath(pathname: string): string {
  // 去掉末尾的 /login、/install 及其后的任意子路径，剩下的就是挂载前缀。
  const m = pathname.match(/^(.*?)\/(?:login|install)(?:\/.*)?$/)
  if (m) return m[1] // 根挂载时这里是 ""（pathname=/login）
  // 兜底：非预期路径（不应发生），按根处理。
  return ''
}

/** 把 config.json 的相对/绝对 base 归一为可直接用于 axios 的形式（保留前导 /）。 */
function normalizeBase(base: string, fallback: string): string {
  const v = (base || '').trim()
  if (!v) return fallback
  return v.startsWith('/') || /^https?:\/\//.test(v) ? v : `/${v}`
}

let current: RuntimeConfig | null = null
let appBase = ''

/** 读取已解析的运行时配置（未初始化前为 null）。 */
export function getRuntimeConfig(): RuntimeConfig | null {
  return current
}

/** IAM 挂载 basePath（如 "" 或 "/iam"）。 */
export function getAppBase(): string {
  return appBase
}

/**
 * 拼接应用内路径（用于整页跳转 `window.location.href`）：`{basePath}{path}`。
 * 例：basePath="/iam"、path="/install" → "/iam/install"；根挂载 → "/install"。
 */
export function appPath(path: string): string {
  const p = path.startsWith('/') ? path : `/${path}`
  return `${appBase}${p}`
}

/**
 * 初始化运行时配置：探测 basePath → 拉 `{basePath}/config.json` → 落定全局配置。
 *
 * 失败时回退为「根路径 + 无前缀」的保守默认（等价于旧行为），保证极端情况下页面仍能渲染，
 * 由后续 API 调用暴露真实错误。
 */
export async function loadRuntimeConfig(): Promise<RuntimeConfig> {
  appBase = detectBasePath(window.location.pathname)
  const fallback: RuntimeConfig = {
    issuer: window.location.origin + '/oidc',
    apiBase: '/v1',
    oidcBase: '/oidc',
    installBase: '/install',
    loginPath: `${appBase}/login`,
    callbackPath: '/auth/callback',
    clientId: '',
  }
  try {
    const res = await fetch(`${appBase}/config.json`, {
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
    })
    if (!res.ok) throw new Error(`config.json HTTP ${res.status}`)
    const raw = (await res.json()) as Partial<RuntimeConfig>
    current = {
      issuer: raw.issuer || fallback.issuer,
      // 后端未下发带前缀 base 时（旧后端），在前端按 basePath 兜底补前缀。
      apiBase: normalizeBase(raw.apiBase || '', `${appBase}/v1`),
      oidcBase: normalizeBase(raw.oidcBase || '', `${appBase}/oidc`),
      installBase: normalizeBase(raw.installBase || '', `${appBase}/install`),
      loginPath: raw.loginPath || fallback.loginPath,
      callbackPath: raw.callbackPath || fallback.callbackPath,
      clientId: raw.clientId || fallback.clientId,
    }
  } catch {
    current = fallback
  }
  return current
}
