import { useState } from 'react'
import { Button, Card, List, Space, Typography, message } from 'antd'
import { useNavigate } from 'react-router'
import { selectTenant } from '../../api/auth'
import { useAuthStore } from '../../stores/authStore'

const SelectTenant = () => {
  const [loadingTenantID, setLoadingTenantID] = useState<number | null>(null)
  const navigate = useNavigate()
  const { personToken, tenants, setTenantSession } = useAuthStore()

  const handleSelectTenant = async (tenantID: number) => {
    if (!personToken) {
      navigate('/login', { replace: true })
      return
    }

    const currentTenant = tenants.find((tenant) => tenant.tenantID === tenantID)
    if (!currentTenant) {
      message.error('租户信息不存在，请重新登录')
      navigate('/login', { replace: true })
      return
    }

    setLoadingTenantID(tenantID)
    try {
      const resp = await selectTenant({ personToken, tenantID })
      setTenantSession({
        tenantToken: resp.data.tenantToken.accessToken,
        refreshToken: resp.data.tenantToken.refreshToken,
        currentTenant,
      })
      navigate('/', { replace: true })
    } catch (error) {
      console.error('选择租户失败:', error)
    } finally {
      setLoadingTenantID(null)
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f2f5',
        padding: 24,
      }}
    >
      <Card title="选择租户" style={{ width: 520, maxWidth: '100%' }}>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Typography.Text type="secondary">
            请选择本次登录要进入的租户。
          </Typography.Text>
          <List
            dataSource={tenants}
            renderItem={(tenant) => (
              <List.Item
                actions={[
                  <Button
                    key={tenant.tenantID}
                    type="primary"
                    loading={loadingTenantID === tenant.tenantID}
                    onClick={() => handleSelectTenant(tenant.tenantID)}
                  >
                    进入租户
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={tenant.name}
                  description={tenant.tag}
                />
              </List.Item>
            )}
          />
        </Space>
      </Card>
    </div>
  )
}

export default SelectTenant
