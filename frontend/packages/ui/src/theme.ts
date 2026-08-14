import type { ThemeConfig } from 'antd'

/**
 * Ark IAM 全局设计令牌
 *
 * 品牌色采用靛蓝渐变（indigo → violet），配合浅灰页面底色与圆角卡片，
 * 统一所有前端应用的视觉语言。
 */
export const brand = {
  primary: '#4f6ef7',
  primaryHover: '#6b86ff',
  primaryActive: '#3a55d6',
  gradient: 'linear-gradient(135deg, #4f6ef7 0%, #7a5af8 55%, #a855f7 100%)',
  gradientSoft: 'linear-gradient(135deg, #eef2ff 0%, #f5f0ff 100%)',
  bg: '#f5f6fa',
  sidebarBg: '#0f172a',
  text: 'rgba(17, 24, 39, 0.88)',
  textSecondary: 'rgba(17, 24, 39, 0.55)',
}

export const themeConfig: ThemeConfig = {
  token: {
    colorPrimary: brand.primary,
    colorInfo: brand.primary,
    colorLink: brand.primary,
    colorSuccess: '#22c55e',
    colorWarning: '#f59e0b',
    colorError: '#ef4444',
    colorBgLayout: brand.bg,
    borderRadius: 8,
    borderRadiusLG: 12,
    colorText: brand.text,
    colorTextSecondary: brand.textSecondary,
    fontFamily:
      "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif",
    boxShadowSecondary: '0 6px 24px rgba(15, 23, 42, 0.08)',
  },
  components: {
    Layout: {
      headerBg: '#ffffff',
      headerHeight: 56,
      siderBg: brand.sidebarBg,
      bodyBg: brand.bg,
    },
    Menu: {
      darkItemBg: brand.sidebarBg,
      darkItemSelectedBg: brand.primary,
      darkItemSelectedColor: '#ffffff',
      darkItemHoverBg: 'rgba(255,255,255,0.08)',
      itemBorderRadius: 8,
      itemHeight: 42,
    },
    Card: {
      headerFontSize: 15,
    },
    Table: {
      headerBg: '#fafbff',
      headerColor: 'rgba(17, 24, 39, 0.65)',
      rowHoverBg: '#f6f8ff',
    },
    Modal: {
      borderRadiusLG: 14,
    },
  },
}
