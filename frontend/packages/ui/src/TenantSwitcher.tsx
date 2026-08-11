import { Dropdown } from 'antd'
import { SwapOutlined } from '@ant-design/icons'
import { useTenantSwitching, getCurrentTenantId } from '@ark-iam/auth'

export function TenantSwitcher() {
  const { tenants, loadTenants, handleSwitchTenant } = useTenantSwitching()

  return (
    <Dropdown
      menu={{
        items: tenants.map((t) => ({
          key: `tenant-${t.tenantID}`,
          label: t.name,
          disabled: String(t.tenantID) === getCurrentTenantId(),
          onClick: () => handleSwitchTenant(t.tenantID),
        })),
      }}
      onOpenChange={(open) => {
        if (open) void loadTenants()
      }}
    >
      <span style={{ color: '#000', cursor: 'pointer', marginRight: 16 }}>
        <SwapOutlined /> 切换租户
      </span>
    </Dropdown>
  )
}
