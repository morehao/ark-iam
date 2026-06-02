import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Descriptions, Table, Button, Modal, Form, Input, DatePicker, message, Tag, Space, Alert } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getOAuthClientDetail, listSecrets, createSecret, deleteSecret, OAuthClientDetail as OAuthClientDetailType, SecretResp, CreateSecretResp } from '../../api/oauthClient'

const OAuthClientDetail = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<OAuthClientDetailType | null>(null)
  const [secrets, setSecrets] = useState<SecretResp[]>([])
  const [loading, setLoading] = useState(false)
  const [secretLoading, setSecretLoading] = useState(false)
  const [secretModalVisible, setSecretModalVisible] = useState(false)
  const [secretForm] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)
  const [createdSecret, setCreatedSecret] = useState<CreateSecretResp | null>(null)

  const fetchDetail = async () => {
    if (!id) return
    setLoading(true)
    try {
      const resp = await getOAuthClientDetail(Number(id))
      setDetail(resp)
    } catch {
      console.error('获取OAuth客户端详情失败')
    } finally {
      setLoading(false)
    }
  }

  const fetchSecrets = async () => {
    if (!id) return
    setSecretLoading(true)
    try {
      const resp = await listSecrets(Number(id))
      setSecrets(resp?.secrets || [])
    } catch {
      console.error('获取密钥列表失败')
    } finally {
      setSecretLoading(false)
    }
  }

  useEffect(() => {
    fetchDetail()
    fetchSecrets()
  }, [id])

  const handleAddSecret = () => {
    setCreatedSecret(null)
    secretForm.resetFields()
    setSecretModalVisible(true)
  }

  const handleDeleteSecret = (record: SecretResp) => {
    Modal.confirm({
      title: '确认删除密钥',
      content: `确定要删除密钥"${record.name}"吗？`,
      onOk: async () => {
        try {
          await deleteSecret(record.id)
          message.success('删除成功')
          fetchSecrets()
        } catch {
          console.error('删除密钥失败')
        }
      },
    })
  }

  const handleSecretSubmit = async () => {
    try {
      const values = await secretForm.validateFields()
      setSubmitLoading(true)
      const resp = await createSecret({
        oauthClientId: Number(id),
        name: values.name,
        expiresAt: values.expiresAt ? values.expiresAt.toISOString() : undefined,
      })
      setCreatedSecret(resp)
      message.success('密钥创建成功，请立即保存密钥明文')
      setSecretModalVisible(false)
      fetchSecrets()
    } catch (error) {
      if (error instanceof Error) {
        console.error('创建密钥失败:', error.message)
      }
    } finally {
      setSubmitLoading(false)
    }
  }

  const statusRender = (val: string) =>
    val === 'enable' ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>

  const typeRender = (val: string) =>
    val === 'third_party' ? <Tag color="orange">第三方</Tag> : <Tag color="blue">第一方</Tag>

  const secretColumns: ColumnsType<SecretResp> = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '前缀', dataIndex: 'valuePrefix', key: 'valuePrefix' },
    { title: '过期时间', dataIndex: 'expiresAt', key: 'expiresAt', render: (val: string | null) => val || '永不过期' },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Button type="link" danger onClick={() => handleDeleteSecret(record)}>删除</Button>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button onClick={() => navigate('/oauthClient')}>返回列表</Button>
      </Space>
      <Card title="客户端基本信息" loading={loading}>
        {detail && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="ID">{detail.oauthClientId}</Descriptions.Item>
            <Descriptions.Item label="租户ID">{detail.tenantId}</Descriptions.Item>
            <Descriptions.Item label="应用ID">{detail.appId}</Descriptions.Item>
            <Descriptions.Item label="客户端ID">{detail.clientID}</Descriptions.Item>
            <Descriptions.Item label="名称">{detail.name}</Descriptions.Item>
            <Descriptions.Item label="类型">{typeRender(detail.type)}</Descriptions.Item>
            <Descriptions.Item label="状态">{statusRender(detail.status)}</Descriptions.Item>
            <Descriptions.Item label="是否第三方">{detail.isThirdParty === 1 ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="授权类型" span={2}>{detail.grantTypes?.join(', ')}</Descriptions.Item>
            <Descriptions.Item label="回调地址" span={2}>{detail.redirectURIs?.join(', ')}</Descriptions.Item>
            <Descriptions.Item label="登出回调" span={2}>{detail.postLogoutRedirectURIs?.join(', ')}</Descriptions.Item>
            <Descriptions.Item label="令牌端点认证方式">{detail.tokenEndpointAuthMethod}</Descriptions.Item>
            <Descriptions.Item label="CORS白名单">{detail.allowedOrigins?.join(', ')}</Descriptions.Item>
            <Descriptions.Item label="强制PKCE">{detail.requirePKCE === 1 ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="需要auth_time">{detail.requireAuthTime === 1 ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="默认Scopes" span={2}>{detail.defaultScopes?.join(', ')}</Descriptions.Item>
            <Descriptions.Item label="AccessToken TTL">{detail.accessTokenTTL}秒</Descriptions.Item>
            <Descriptions.Item label="RefreshToken TTL">{detail.refreshTokenTTL}秒</Descriptions.Item>
            <Descriptions.Item label="创建时间" span={2}>{detail.createdAt}</Descriptions.Item>
          </Descriptions>
        )}
      </Card>

      <Card title="密钥管理" style={{ marginTop: 16 }}>
        <div style={{ marginBottom: 16 }}>
          <Button type="primary" onClick={handleAddSecret}>新建密钥</Button>
        </div>
        {createdSecret && (
          <Alert
            type="warning"
            showIcon
            message="密钥创建成功"
            description={
              <div>
                <p><strong>密钥明文（请立即保存，关闭后将不再显示）：</strong></p>
                <Input.TextArea
                  rows={2}
                  value={createdSecret.secret}
                  readOnly
                  style={{ fontFamily: 'monospace' }}
                />
              </div>
            }
            closable
            onClose={() => setCreatedSecret(null)}
            style={{ marginBottom: 16 }}
          />
        )}
        <Table
          columns={secretColumns}
          dataSource={secrets}
          rowKey="id"
          loading={secretLoading}
          pagination={false}
        />
      </Card>

      <Modal
        title="新建密钥"
        open={secretModalVisible}
        onOk={handleSecretSubmit}
        onCancel={() => setSecretModalVisible(false)}
        confirmLoading={submitLoading}
      >
        <Form form={secretForm} layout="vertical">
          <Form.Item name="name" label="密钥名称" rules={[{ required: true, message: '请输入密钥名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="expiresAt" label="过期时间">
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default OAuthClientDetail
