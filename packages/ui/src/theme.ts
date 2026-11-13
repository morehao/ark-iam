import type { ThemeConfig } from 'antd'

/**
 * Ark IAM 全局设计令牌（唯一事实源）
 *
 * 设计方向：Ant Design 原生（对齐同级仓库 ark-go-starter）。蓝色主色 #1677ff、
 * 浅色 Sider（白底）+ 浅色 Layout、6px 控件圆角 / 8px 卡片圆角；页面表面中性灰阶、
 * 卡片 hairline 边框分层、无默认阴影（阴影仅留给 Modal / Dropdown 等浮层）。
 *
 * 视觉规范全文见根目录 DESIGN.md（唯一设计事实源，遵循 Google DESIGN.md 官方格式规范，
 * 可用 `npx @google/design.md lint DESIGN.md` 校验）。
 * 本文件 tokens 与 DESIGN.md front-matter 的令牌一一对应，改动需两处同步。
 */
export const tokens = {
  // 品牌 Brand：蓝色，仅用于主按钮 / 链接 / 选中态 / 分页当前页 / 聚焦环
  primary: '#1677ff',
  primaryHover: '#4096ff',
  primaryActive: '#0958d9',
  // 主色「承载白字」的无障碍变体（#1677ff 白字仅 4.10:1，此处 5.57:1 满足 WCAG AA）
  primaryStrong: '#0b63d6',
  primarySoft: '#e6f4ff', // 选中 Tag / 批量操作提示条 / 菜单选中项浅蓝底
  primarySoftBorder: '#91caff', // 上者描边
  gradient: 'linear-gradient(135deg, #1677ff 0%, #4096ff 55%, #69b1ff 100%)',
  gradientSoft: 'linear-gradient(135deg, #e6f4ff 0%, #f0f8ff 100%)',
  purple: '#69b1ff', // 分类点缀强调（AntD 蓝系延续）
  selectedBg: '#e6f4ff', // 浅色导航选中项浅蓝底

  // 表面 Surface（中性灰阶）
  bg: '#f5f5f5', // 页面布局底色
  cardBg: '#ffffff',
  sidebarBg: '#ffffff', // 浅色 Sider
  sidebarSubBg: '#fafafa', // 侧栏次级/折叠底
  headerBg: '#ffffff',
  tableHeaderBg: '#fafafa',
  rowHoverBg: '#fafafa',
  softFill: '#f5f5f5', // 中性填充底（菜单 hover 等）
  codeBg: '#fafafa',

  // 边框 Border
  border: '#f0f0f0', // 卡片 / 分隔线
  borderStrong: '#d9d9d9', // 控件描边（输入框 / 按钮 / 分页器）

  // 文字 Text（antd 透明度体系；保留三级 key + 新增 tertiary）
  text: 'rgba(0,0,0,0.88)',
  textSecondary: 'rgba(0,0,0,0.65)',
  textTertiary: 'rgba(0,0,0,0.45)',
  textPlaceholder: 'rgba(0,0,0,0.25)',

  // 语义色 Semantic（与 antd colorSuccess/Warning/Error 对齐，仅状态 Tag/告警使用）
  success: '#52c41a',
  warning: '#faad14',
  error: '#ff4d4f',
  warningBg: '#fffbe6', // 轻量告警底色
  warningBorder: '#ffe58f', // 轻量告警描边

  // 圆角 Radius（对齐 DESIGN.md rounded 段：sm 4 / md 6 / lg 8）
  radiusSm: 4, // Tag / 小徽标
  radiusMd: 6, // 控件（按钮 / 输入 / 分页器 / 菜单项）
  radiusLg: 8, // 卡片 / 内容卡 / Statistic 卡 / Dropdown / Modal

  // 字号 Font size（对齐 DESIGN.md typography 段：body 14 / small 13 / tag 12）
  fontSizeBody: 14,
  fontSizeSmall: 13,
  fontSizeTag: 12,
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
    colorBgContainer: tokens.cardBg,
    colorBorderSecondary: tokens.border,
    colorBorder: tokens.borderStrong,
    borderRadius: tokens.radiusMd,
    borderRadiusLG: tokens.radiusLg,
    colorText: tokens.text,
    colorTextSecondary: tokens.textSecondary,
    colorTextTertiary: tokens.textTertiary,
    colorTextQuaternary: tokens.textPlaceholder,
    controlHeight: 32,
    fontFamily:
      "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif",
    fontSize: tokens.fontSizeBody,
    boxShadowSecondary:
      '0 6px 16px rgba(0,0,0,0.08), 0 3px 6px -4px rgba(0,0,0,0.12), 0 9px 28px 8px rgba(0,0,0,0.05)', // 仅浮层/Modal 使用
  },
  components: {
    Layout: {
      headerBg: tokens.headerBg,
      headerHeight: 64,
      siderBg: tokens.sidebarBg,
      bodyBg: tokens.bg,
    },
    Menu: {
      darkItemBg: tokens.sidebarBg,
      darkItemSelectedBg: tokens.primarySoft,
      darkItemSelectedColor: tokens.primaryActive,
      darkItemHoverBg: tokens.softFill,
      itemBorderRadius: tokens.radiusMd,
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
      borderRadiusLG: tokens.radiusLg,
    },
  },
}
