import type { ReactNode } from 'react'
import {
  ApartmentOutlined,
  ApiOutlined,
  AppstoreAddOutlined,
  AppstoreOutlined,

  AuditOutlined,
  BankOutlined,
  DashboardOutlined,
  GlobalOutlined,
  KeyOutlined,
  ProfileOutlined,
  ProjectOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons'

/**
 * 菜单图标注册表（唯一事实源）—— icon key → 语义唯一图标。
 *
 * 背景：菜单 icon 是后端 `menu.icon` 里的字符串，前端按 key 映射组件。历史上各 App 各自
 * 硬编码 `ICON_MAP`，导致「一 key 多义 / 多菜单共用图标」（如租户管理与自定义域名都用
 * global）。这里把映射收敛为单一注册表：**一 key 一图标、语义唯一**，供侧边栏渲染
 * （菜单 icon）、菜单管理表单（icon 下拉选项）共用。
 *
 * 约定：
 * - key 是稳定标识，落库存的是 key（不是 React 组件名）。
 * - 未注册的 key 由 `menuIcon()` 回落兜底图标（前端强约束 + 后端宽容兜底，不锁死运维自定义）。
 * - P3 语义化改名后，种子/存量库使用「新 key」列；旧 key 保留以兼容未补录的存量库（灰度期）。
 */

/** 平台管理后台菜单图标（对象域语义唯一）。 */
const PLATFORM_ICONS: Record<string, ReactNode> = {
  // —— 语义化新 key（P3 起种子使用）——
  'tenant-group': <BankOutlined />, // 租户中心（目录）
  tenant: <TeamOutlined />, // 租户管理（组织群体）
  'app-subscribe': <AppstoreAddOutlined />, // 租户应用（订阅/开通）
  domain: <GlobalOutlined />, // 自定义域名（独占 globe）
  'app-group': <AppstoreOutlined />, // 应用中心（目录）
  application: <ProjectOutlined />, // 应用管理（与目录区分）
  oauth: <ApiOutlined />, // OAuth 客户端（接入接口）
  menu: <ProfileOutlined />, // 菜单管理（清单/字典）
  dashboard: <DashboardOutlined />, // 工作台
  setting: <SettingOutlined />, // 平台管理（目录）
  audit: <AuditOutlined />, // 审计日志

  // —— 旧 key 兼容（P3 前的存量库 / 灰度期）——
  apartment: <BankOutlined />, // 旧「租户中心目录」
  global: <GlobalOutlined />, // 旧「租户管理 / 自定义域名」（曾重复）
  shopping: <AppstoreAddOutlined />, // 旧「租户应用」
  app: <AppstoreOutlined />, // 旧「应用中心 / 应用管理」
  key: <KeyOutlined />, // 旧「OAuth 客户端」
}

/** 租户管理后台菜单图标。 */
const TENANT_ICONS: Record<string, ReactNode> = {
  department: <ApartmentOutlined />, // 部门管理（组织树）
  user: <UserOutlined />, // 用户管理
  role: <SafetyCertificateOutlined />, // 角色管理（授权凭证）
  apikey: <KeyOutlined />, // API 密钥（真正的钥匙）
  // 旧 key 兼容
  key: <KeyOutlined />, // 旧「API 密钥」
}

/**
 * 全量菜单图标注册表（含平台端与租户端全部 key，供表单下拉遍历）。
 *
 * 注意：平台端与租户端存在同名 key（如 `user`、`role` 仅租户端；`key` 两端都曾有），
 * 这里以「合并全集」对外暴露；同名时后合并者覆盖，但当前两端无冲突语义的同名 key。
 */
export const MENU_ICONS: Record<string, ReactNode> = {
  ...PLATFORM_ICONS,
  ...TENANT_ICONS,
}

/** 兜底图标：未注册 key 时使用。 */
const FALLBACK_ICON: ReactNode = <AppstoreOutlined />

/**
 * 取菜单图标。未注册（含空串/undefined）时回落兜底图标，绝不抛错、绝不空白。
 *
 * @param key 后端 `menu.icon` 的字符串值
 */
export function menuIcon(key?: string): ReactNode {
  if (!key) return FALLBACK_ICON
  return MENU_ICONS[key] ?? FALLBACK_ICON
}

/**
 * 菜单图标 key → 中文名（供菜单管理表单 Select 的选项文案，便于运维辨认）。
 * 未列的 key 回退用 key 本身作展示文案。
 */
export const MENU_ICON_LABELS: Record<string, string> = {
  'tenant-group': '租户中心',
  tenant: '租户',
  'app-subscribe': '租户应用',
  domain: '域名',
  'app-group': '应用中心',
  application: '应用',
  oauth: 'OAuth 客户端',
  menu: '菜单',
  dashboard: '工作台',
  setting: '设置',
  audit: '审计',
  department: '部门',
  user: '用户',
  role: '角色',
  apikey: 'API 密钥',
}

/** 菜单图标 key 的可选项（供菜单管理表单 Select 使用），按 key 稳定排序，label 带中文名。 */
export function menuIconOptions(): { value: string; label: string }[] {
  return Object.keys(MENU_ICONS)
    .sort()
    .map((key) => ({ value: key, label: MENU_ICON_LABELS[key] ? `${MENU_ICON_LABELS[key]}（${key}）` : key }))
}
