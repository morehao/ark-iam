import { Dropdown, Tag } from 'antd'
import { SwapOutlined } from '@ant-design/icons'
import { useTenantSwitching, getCurrentTenantId } from '@ark-iam/auth'

export function TenantSwitcher() {
  const { tenants, loadTenants, handleSwitchTenant } = useTenantSwitching()
  const currentId = getCurrentTenantId()
  const currentName = tenants.find((t) => String(t.tenantID) === currentId)?.name || '当前租户'

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
          color: '#333',
          cursor: 'pointer',
          fontSize: 13,
          lineHeight: 1,
          display: 'inline-flex',
          alignItems: 'center',
          height: 32,
          padding: '0 12px',
          boxSizing: 'border-box',
          borderRadius: 8,
          border: '1px solid #e5e7eb',
          background: '#fafbff',
          transition: 'all 0.2s',
        }}
        className="tenant-switcher"
      >
        <SwapOutlined style={{ color: '#4f6ef7' }} />
        <span style={{ fontWeight: 500, lineHeight: 1 }}>{currentName}</span>
      </span>
    </Dropdown>
  )
}
