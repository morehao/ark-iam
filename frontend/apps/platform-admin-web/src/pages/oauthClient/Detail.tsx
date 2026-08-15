import { useCallback, useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Alert, Button, Card, DatePicker, Descriptions, Form, Input, Modal, Popconfirm, Space, Spin, Table, message } from 'antd'
import { ArrowLeftOutlined, PlusOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { createOAuthSecret, deleteOAuthSecret, getOAuthClientDetail, listOAuthSecrets } from '@ark-iam/api'
import type { OAuthClientDetail as OAuthClientDetailType, OAuthSecretCreateResp, OAuthSecretItem } from '@ark-iam/types'
import { fmtTime, StatusTag, TypeTag } from '../../components/common'
import { IDCell } from '@ark-iam/ui'

export default function OAuthClientDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<OAuthClientDetailType | null>(null)
  const [loading, setLoading] = useState(false)
  const [secrets, setSecrets] = useState<OAuthSecretItem[]>([])
  const [secretLoading, setSecretLoading] = useState(false)
  const [secretModalOpen, setSecretModalOpen] = useState(false)
  const [secretForm] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)
  const [createdSecret, setCreatedSecret] = useState<OAuthSecretCreateResp | null>(null)

  const clientID = id ?? ""

  const fetchAll = useCallback(async () => {
    if (!id) return
    setLoading(true)
    try {
      const resp = await getOAuthClientDetail(clientID)
      setDetail(resp)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [id, clientID])

  const fetchSecrets = useCallback(async () => {
    if (!id) return
    setSecretLoading(true)
    try {
      const resp = await listOAuthSecrets(clientID)
      setSecrets(resp?.secrets || [])
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSecretLoading(false)
    }
  }, [id, clientID])

  useEffect(() => {
    void fetchAll()
    void fetchSecrets()
  }, [fetchAll, fetchSecrets])

  const handleCreateSecret = async () => {
    try {
      const values = await secretForm.validateFields()
      setSubmitLoading(true)
      const resp = await createOAuthSecret({
        applicationClientID: clientID,
        name: values.name,
        expiresAt: values.expiresAt ? values.expiresAt.toISOString() : undefined,
      })
      setCreatedSecret(resp)
      message.success('密钥创建成功，请立即保存明文')
      setSecretModalOpen(false)
      secretForm.resetFields()
      void fetchSecrets()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const secretColumns: ColumnsType<OAuthSecretItem> = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '名称', dataIndex: 'name', key: 'name' },
    {
      title: '前缀',
      dataIndex: 'valuePrefix',
      key: 'valuePrefix',
      render: (v: string) => <span style={{ fontFamily: 'monospace' }}>{v || '-'}</span>,
    },
    { title: '过期时间', dataIndex: 'expiresAt', key: 'expiresAt', width: 180, render: (v: string | null) => v || '永不过期' },
    { title: '创建时间', key: 'createdAt', width: 170, render: (_, r) => fmtTime(r.createdAt) },
    {
      title: '操作',
      key: 'action',
      width: 90,
      render: (_, r) => (
        <Popconfirm
          title="确认删除该密钥？"
          onConfirm={async () => {
            try {
              await deleteOAuthSecret(clientID, r.id)
              message.success('删除成功')
              void fetchSecrets()
            } catch {
              /* 拦截器已提示 */
            }
          }}
        >
          <Button type="link" size="small" danger>
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ]

  if (loading && !detail) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: 80 }}>
        <Spin size="large" />
      </div>
    )
  }

  if (!detail) return null

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/oauthClient')}>
          返回列表
        </Button>
      </Space>

      <Card title="基本信息" style={{ borderRadius: 12, marginBottom: 16, border: '1px solid #f0f0f0' }} styles={{ body: { padding: 24 } }}>
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="ID"><IDCell value={detail.applicationClientID} /></Descriptions.Item>
          <Descriptions.Item label="租户ID"><IDCell value={detail.tenantID} /></Descriptions.Item>
          <Descriptions.Item label="所属应用ID"><IDCell value={detail.appID} /></Descriptions.Item>
          <Descriptions.Item label="客户端ID">
            <IDCell value={detail.clientID} />
          </Descriptions.Item>
          <Descriptions.Item label="名称">{detail.name || '-'}</Descriptions.Item>
          <Descriptions.Item label="类型">
            <TypeTag value={detail.type} />
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <StatusTag value={detail.status} />
          </Descriptions.Item>
          <Descriptions.Item label="是否第三方">{detail.isThirdParty === 1 ? '是' : '否'}</Descriptions.Item>
          <Descriptions.Item label="令牌端点认证方式">{detail.tokenEndpointAuthMethod || '-'}</Descriptions.Item>
          <Descriptions.Item label="创建时间">{fmtTime(detail.createdAt)}</Descriptions.Item>
          <Descriptions.Item label="授权类型" span={2}>
            {detail.grantTypes?.join(', ') || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="回调地址" span={2}>
            {detail.redirectURIs?.join(', ') || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="登出回调地址" span={2}>
            {detail.postLogoutRedirectURIs?.join(', ') || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="后端登出地址">{detail.backChannelLogoutURI || '-'}</Descriptions.Item>
          <Descriptions.Item label="响应类型">{detail.responseTypes?.join(', ') || '-'}</Descriptions.Item>
          <Descriptions.Item label="CORS 白名单" span={2}>
            {detail.allowedOrigins?.join(', ') || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="强制 PKCE">{detail.requirePKCE === 1 ? '是' : '否'}</Descriptions.Item>
          <Descriptions.Item label="需要 auth_time">{detail.requireAuthTime === 1 ? '是' : '否'}</Descriptions.Item>
          <Descriptions.Item label="默认 Scopes" span={2}>
            {detail.defaultScopes?.join(', ') || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="AccessToken TTL">{detail.accessTokenTTL} 秒</Descriptions.Item>
          <Descriptions.Item label="RefreshToken TTL">{detail.refreshTokenTTL} 秒</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="密钥管理" style={{ borderRadius: 12, border: '1px solid #f0f0f0' }} styles={{ body: { padding: 24 } }}>
        <Space style={{ marginBottom: 16 }}>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setCreatedSecret(null)
              secretForm.resetFields()
              setSecretModalOpen(true)
            }}
          >
            新建密钥
          </Button>
        </Space>
        {createdSecret && (
          <Alert
            type="warning"
            showIcon
            message="密钥创建成功"
            description={
              <div>
                <div style={{ marginBottom: 8 }}>密钥明文，请立即保存，关闭后不再显示：</div>
                <Input.TextArea rows={2} value={createdSecret.secret} readOnly style={{ fontFamily: 'monospace' }} />
              </div>
            }
            closable
            onClose={() => setCreatedSecret(null)}
            style={{ marginBottom: 16 }}
          />
        )}
        <Table<OAuthSecretItem>
          rowKey="id"
          columns={secretColumns}
          dataSource={secrets}
          loading={secretLoading}
          pagination={false}
        />
      </Card>

      <Modal
        title="新建密钥"
        open={secretModalOpen}
        onOk={() => void handleCreateSecret()}
        onCancel={() => setSecretModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={480}
      >
        <Form form={secretForm} layout="vertical">
          <Form.Item name="name" label="密钥名称" rules={[{ required: true, message: '请输入密钥名称' }]}>
            <Input placeholder="用于标识密钥用途" />
          </Form.Item>
          <Form.Item name="expiresAt" label="过期时间">
            <DatePicker showTime style={{ width: '100%' }} placeholder="选填，不填则永不过期" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
