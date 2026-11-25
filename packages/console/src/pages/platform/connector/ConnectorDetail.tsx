import { useEffect, useState } from 'react'
import { Button, Descriptions, Drawer, Space, Tag } from 'antd'
import { EditOutlined } from '@ant-design/icons'
import { fmtTime, IDCell } from '@ark-iam/ui'
import { getConnectorDetail } from '@ark-iam/api'
import type { ConnectorItem } from '@ark-iam/types'

interface Props {
  open: boolean
  connector: ConnectorItem | null
  onClose: () => void
  onEdit: (c: ConnectorItem) => void
}

function FlagRow({ value }: { value: string }) {
  return value === 'enable' ? (
    <Tag color="success" variant="filled">
      开启
    </Tag>
  ) : (
    <Tag variant="filled">关闭</Tag>
  )
}

export default function ConnectorDetail({ open, connector, onClose, onEdit }: Props) {
  const [detail, setDetail] = useState<ConnectorItem | null>(connector)

  useEffect(() => {
    if (!open || !connector) return
    setDetail(connector)
    // 拉最新详情（列表接口 config 的 clientSecret 已脱敏，详情同为脱敏值）
    getConnectorDetail(connector.connectorID)
      .then((resp) => resp && setDetail(resp))
      .catch(() => {
        /* 拦截器已提示 */
      })
  }, [open, connector])

  if (!detail) return null
  const c = detail.config ?? {}

  return (
    <Drawer
      title={`连接器详情 · ${detail.displayName}`}
      width={640}
      open={open}
      onClose={onClose}
      extra={
        <Space>
          <Button type="primary" icon={<EditOutlined />} onClick={() => onEdit(detail)}>
            编辑
          </Button>
        </Space>
      }
    >
      <Descriptions column={1} bordered size="small">
        <Descriptions.Item label="连接器 ID">
          <IDCell value={detail.connectorID} />
        </Descriptions.Item>
        <Descriptions.Item label="显示名称">{detail.displayName}</Descriptions.Item>
        <Descriptions.Item label="标识名">
          <code>{detail.name}</code>
        </Descriptions.Item>
        <Descriptions.Item label="提供商">{detail.provider}</Descriptions.Item>
        <Descriptions.Item label="协议">{detail.protocol.toUpperCase()}</Descriptions.Item>
        <Descriptions.Item label="状态">
          {detail.status === 'enable' ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>}
        </Descriptions.Item>
        <Descriptions.Item label="默认落租户">
          {detail.defaultTenantName || detail.defaultTenantID || '-'}
          {detail.defaultTenantName && detail.defaultTenantID && (
            <span style={{ color: '#999', marginLeft: 8, fontSize: 12 }}>{detail.defaultTenantID}</span>
          )}
        </Descriptions.Item>
      </Descriptions>

      <div style={{ margin: '20px 0 8px', fontWeight: 600 }}>协议配置</div>
      <Descriptions column={1} bordered size="small">
        {c.issuer && <Descriptions.Item label="Issuer">{c.issuer}</Descriptions.Item>}
        {c.tenant && <Descriptions.Item label="Tenant（协议）">{c.tenant}</Descriptions.Item>}
        {c.authUrl && <Descriptions.Item label="Auth URL">{c.authUrl}</Descriptions.Item>}
        {c.tokenUrl && <Descriptions.Item label="Token URL">{c.tokenUrl}</Descriptions.Item>}
        {c.userInfoUrl && <Descriptions.Item label="UserInfo URL">{c.userInfoUrl}</Descriptions.Item>}
        <Descriptions.Item label="Client ID">{c.clientID || '-'}</Descriptions.Item>
        <Descriptions.Item label="Client Secret">
          <Space>
            <span style={{ letterSpacing: 1 }}>{c.clientSecret || '-'}</span>
            <Tag variant="filled" color="gold">
              已加密存储
            </Tag>
          </Space>
        </Descriptions.Item>
        <Descriptions.Item label="回调地址">{c.redirectUri || '-'}</Descriptions.Item>
        <Descriptions.Item label="Scope">
          {c.scopes?.length ? (
            <Space wrap size={4}>
              {c.scopes.map((s) => (
                <Tag key={s} variant="filled">
                  {s}
                </Tag>
              ))}
            </Space>
          ) : (
            '-'
          )}
        </Descriptions.Item>
      </Descriptions>

      <div style={{ margin: '20px 0 8px', fontWeight: 600 }}>声明映射</div>
      <Descriptions column={2} bordered size="small">
        <Descriptions.Item label="Subject">{detail.claimMapping?.subject || '默认'}</Descriptions.Item>
        <Descriptions.Item label="Email">{detail.claimMapping?.email || '默认'}</Descriptions.Item>
        <Descriptions.Item label="Username">{detail.claimMapping?.username || '默认'}</Descriptions.Item>
        <Descriptions.Item label="DisplayName">{detail.claimMapping?.displayName || '默认'}</Descriptions.Item>
        <Descriptions.Item label="AvatarUrl" span={2}>
          {detail.claimMapping?.avatarUrl || '默认'}
        </Descriptions.Item>
      </Descriptions>

      <div style={{ margin: '20px 0 8px', fontWeight: 600 }}>域策略与开关</div>
      <Descriptions column={1} bordered size="small">
        <Descriptions.Item label="允许的邮箱域">
          {detail.domainPolicy?.allowedDomains?.length ? (
            <Space wrap size={4}>
              {detail.domainPolicy.allowedDomains.map((d) => (
                <Tag key={d} variant="filled" color="green">
                  {d}
                </Tag>
              ))}
            </Space>
          ) : (
            '不限制'
          )}
        </Descriptions.Item>
        <Descriptions.Item label="拒绝的邮箱域">
          {detail.domainPolicy?.blockedDomains?.length ? (
            <Space wrap size={4}>
              {detail.domainPolicy.blockedDomains.map((d) => (
                <Tag key={d} variant="filled" color="red">
                  {d}
                </Tag>
              ))}
            </Space>
          ) : (
            '无'
          )}
        </Descriptions.Item>
        <Descriptions.Item label="自动创建用户">
          <FlagRow value={detail.allowAutoCreateUser} />
        </Descriptions.Item>
        <Descriptions.Item label="资料同步">
          <FlagRow value={detail.syncProfile} />
        </Descriptions.Item>
        <Descriptions.Item label="令牌存储">
          <FlagRow value={detail.enableTokenStorage} />
        </Descriptions.Item>
        <Descriptions.Item label="创建时间">{fmtTime(detail.createdAt)}</Descriptions.Item>
        <Descriptions.Item label="更新时间">{fmtTime(detail.updatedAt)}</Descriptions.Item>
      </Descriptions>
    </Drawer>
  )
}
