import type { ThemeConfig } from 'antd'

/**
 * Ark IAM 全局设计令牌（唯一事实源）
 *
 * 设计方向：「冷白工程台」收敛（受 Stripe / Vercel 工程控制台启发，见 DESIGN.md 参照锚点）。
 * 靛蓝只出现在主按钮 / 链接 / 选中态 / 品牌区；页面表面一律中性灰阶，
 * 卡片以 hairline 边框分层（无阴影）；文字为三级 hex 灰阶。
 * 命名与 frontend/DESIGN.md 的 colors 段一一对应，改动需两处同步。
 */
export const tokens = {
  // 品牌 Brand：靛蓝 → 紫，仅用于主按钮 / 链接 / 选中态 / 登录与 Logo 品牌区
  primary: '#4f6ef7',
  primaryHover: '#6b86ff',
  primaryActive: '#3a55d6',
  gradient: 'linear-gradient(135deg, #4f6ef7 0%, #7a5af8 55%, #a855f7 100%)',
  gradientSoft: 'linear-gradient(135deg, #eef2ff 0%, #f5f0ff 100%)',
  purple: '#7a5af8', // 品牌紫：分类图标/点缀强调（极少量）
  selectedBg: 'rgba(79, 110, 247, 0.14)', // 深色导航选中项柔和底（非整块亮色）

  // 表面 Surface（中性灰阶，禁用靛蓝淡底 tint）
  bg: '#f6f7f9', // 页面布局底色（冷白）
  cardBg: '#ffffff',
  sidebarBg: '#0f172a',
  headerBg: '#ffffff',
  tableHeaderBg: '#f7f8fa',
  rowHoverBg: '#f3f4f6',
  softFill: '#f1f3f5', // 中性填充底（统计卡图标底、应用固定条等）
  codeBg: '#fafafa', // 代码/内容块底色

  // 边框 Border（冷调 hairline，卡片分层的主语言）
  border: '#e6e9ef',
  borderStrong: '#d3d8e0', // 更强的分割线（Header 内分割线等）

  // 文字 Text（三级 hex 灰阶）
  text: '#1f2430',
  textSecondary: '#64748d', // 借鉴 Stripe ink-mute
  textPlaceholder: '#94a3b8', // 空态/占位辅助文字

  // 语义色 Semantic（与 antd colorSuccess/Warning/Error 对齐，仅状态 Tag/告警使用）
  success: '#22c55e',
  warning: '#f59e0b',
  error: '#ef4444',
  warningBg: '#fffbe6', // 轻量告警底色
  warningBorder: '#ffe58f', // 轻量告警描边
}

/** 历史兼容：品牌相关令牌（主色/渐变/背景/文字）。新代码请引用 tokens。 */
export const brand = {
  primary: tokens.primary,
  primaryHover: tokens.primaryHover,
  primaryActive: tokens.primaryActive,
  gradient: tokens.gradient,
  gradientSoft: tokens.gradientSoft,
  bg: tokens.bg,
  sidebarBg: tokens.sidebarBg,
  text: tokens.text,
  textSecondary: tokens.textSecondary,
}

export const themeConfig: ThemeConfig = {
  token: {
    colorPrimary: tokens.primary,
    colorInfo: tokens.primary,
    colorLink: tokens.primary,
    colorSuccess: tokens.success,
    colorWarning: tokens.warning,
    colorError: tokens.error,
    colorBgLayout: tokens.bg,
    colorBorderSecondary: tokens.border, // hairline 统一为冷调
    borderRadius: 8,
    borderRadiusLG: 12,
    colorText: tokens.text,
    colorTextSecondary: tokens.textSecondary,
    fontFamily:
      "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif",
    boxShadowSecondary: '0 6px 24px rgba(15, 23, 42, 0.08)', // 仅浮层/Modal 使用
  },
  components: {
    Layout: {
      headerBg: tokens.headerBg,
      headerHeight: 56,
      siderBg: tokens.sidebarBg,
      bodyBg: tokens.bg,
    },
    Menu: {
      darkItemBg: tokens.sidebarBg,
      darkItemSelectedBg: tokens.selectedBg,
      darkItemSelectedColor: '#ffffff',
      darkItemHoverBg: 'rgba(255,255,255,0.08)',
      itemBorderRadius: 8,
      itemHeight: 42,
    },
    Card: {
      headerFontSize: 15,
    },
    Table: {
      headerBg: tokens.tableHeaderBg,
      headerColor: tokens.textSecondary,
      rowHoverBg: tokens.rowHoverBg,
    },
    Modal: {
      borderRadiusLG: 14,
    },
  },
}
