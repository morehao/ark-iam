export const CONFIG = {
  issuer: 'http://localhost:8100/oidc',
  rp1Url: 'http://localhost:3002/',
  loginWebUrl: 'http://localhost:3000/login',
  platformAdminUrl: 'http://localhost:3001/',
  identifier: 'admin',
  // 管理员口令由初始化页面指定（首次引导时经 POST /install/initialize 写入）。
  // 必须满足服务端 credential.ValidateStrength：含大写字母、小写字母与数字。
  // 启动期播种已删除，因此这个值只在 global-setup 的首次引导里生效；
  // 库已初始化时不会再改写（这是自锁语义，不是幂等 upsert）。
  password: 'Admin123',
  // 初始化引导的一次性令牌：与后端 BOOTSTRAP_TOKEN 环境变量一致。
  // 后端未配置该变量时 /install/initialize 整体不可用（fail-closed）。
  bootstrapToken: 'e2e-bootstrap-token',
  // 初始化接口地址（gateway 聚合部署，与 issuer 同源）。
  installBaseURL: 'http://localhost:8100',
  sessionTTL: 30,
  authRequestTTL: 30,
  authCodeTTL: 15,
};
