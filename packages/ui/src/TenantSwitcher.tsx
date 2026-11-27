import { useEffect, useState } from 'react'
import { Dropdown, Tag, Spin } from 'antd'
import { DownOutlined, SwapOutlined } from '@ant-design/icons'
import { getUserinfo } from '@ark-iam/api'
import { useTenantSwitching, getCurrentTenantId } from '@ark-iam/auth'
import { tokens } from './theme'

export function TenantSwitcher() {
  const { tenants, loadTenants, handleSwitchTenant } = useTenantSwitching()
  const [hovered, setHovered] = useState(false)
  // 当前租户 ID 以 /auth/userinfo 为准：该值由后端从登录 token 上下文判定，
  // 对多租户用户同样成立（id_token 的 tenant_id claim 仅单租户才产出，不能作为来源）。
  // 切换租户后置为「重定向中」，用 setCurrentTenantId 写入的模块变量即时回显。
  const [currentId, setCurrentId] = useState(getCurrentTenantId())
  const currentName = tenants.find((t) => String(t.tenantID) === currentId)?.name

  useEffect(() => {
    void loadTenants()
    let alive = true
    getUserinfo()
      .then((resp) => {
        const id = resp?.userInfo?.tenantID
        if (alive && id) setCurrentId(String(id))
      })
      .catch(() => {
        // 拿不到时保持模块变量兜底（切换后由 setCurrentTenantId 写入），不阻断展示。
      })
    return () => {
      alive = false
    }
  }, [loadTenants])

  // 名称已解析 → 显示名称；列表尚为空 → 转圈；列表已加载但未匹配到当前租户 → 引导选择。
  const label = currentName ?? (tenants.length > 0 ? '选择租户' : null)

  return (
    <Dropdown
      menu={{
        items: tenants.map((t) => ({
          key: `tenant-${t.tenantID}`,
          label: (
            <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              {t.name}
              {String(t.tenantID) === currentId && <Tag color="blue" style={{ marginInlineEnd: 0 }}>当前</Tag>}
            </span>
          ),
          disabled: String(t.tenantID) === currentId,
          onClick: () => handleSwitchTenant(t.tenantID),
        })),
      }}
      onOpenChange={(open) => {
        if (open) void loadTenants()
      }}
    >
      <span
        style={{
          color: tokens.text,
          cursor: 'pointer',
          fontSize: 13,
          lineHeight: 1,
          display: 'inline-flex',
          alignItems: 'center',
          gap: 7,
          height: 32,
          padding: '0 8px',
          boxSizing: 'border-box',
          borderRadius: tokens.radiusMd,
          background: hovered ? tokens.softFill : 'transparent',
          transition: 'background 0.15s',
        }}
        className="tenant-switcher"
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      >
        <SwapOutlined style={{ color: tokens.primary }} />
        {label ? (
          <>
            <span style={{ fontWeight: 500, lineHeight: 1 }}>{label}</span>
            <DownOutlined style={{ fontSize: 10, color: tokens.textTertiary }} />
          </>
        ) : (
          <Spin size="small" />
        )}
      </span>
    </Dropdown>
  )
}
