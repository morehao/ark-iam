/**
 * /install 页面全部用户可见文案的唯一出处。
 *
 * 集中在此的原因：同一句"服务端未配置 BOOTSTRAP_TOKEN"会出现在两个入口——
 * GET /install/status 的 tokenRequired=false，以及 POST /install/initialize 的 107002；
 * 两处必须逐字一致，否则运维会以为是两个不同故障。
 */
export const INSTALL_MESSAGES = {
  /** 已初始化终态（GET status.initialized=true，或 POST initialize 返回 107000） */
  alreadyInitialized: '系统已完成初始化，无需重复初始化。可直接从下面的控制台入口登录。',
  /** 服务端未配置 BOOTSTRAP_TOKEN（status.tokenRequired=false 或错误码 107002）：必须给出可执行的下一步 */
  tokenNotConfigured:
    '服务端未配置 BOOTSTRAP_TOKEN 环境变量，初始化接口整体不可用。请在后端进程中设置该环境变量并重启后端，然后回到本页重试。',
  /** 数据库缺表（status.schemaReady=false） */
  schemaNotReady:
    '数据库尚未就绪（缺少数据表）。后端可能以 db.auto_migrate: false 启动，请先按部署文档完成数据库迁移，再回到本页重试。',
  /** 107001 */
  tokenInvalid: '初始化令牌不正确',
  /** 107003 */
  initializeFailed: '初始化执行失败，请检查后端日志后重试（已保留你填写的表单内容）',
  /** 107004：业务接口守卫（登录侧据此跳 /install） */
  pending: '系统尚未初始化，请先完成初始化',
  /** 107005 */
  badRequest: '初始化参数不合法，请检查表单后重试',
  /** 107006 */
  passwordWeak: '管理员密码强度不足：须包含大写字母、小写字母与数字，长度 8~128 个字符',
  /** 未知错误码兜底 */
  unknown: '初始化失败，请检查后端日志后重试',
  /** 状态探测失败（网络/端点未部署） */
  statusUnavailable: '无法获取初始化状态，请确认后端服务已启动后重试。',
} as const
