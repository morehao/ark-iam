import { useCallback, useEffect, useState } from 'react'
import { Table, Button, Space, Input, Modal, Form, Select, Popconfirm, message } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { PageContainer } from '@ark-iam/ui'
import { createOAuthClient, deleteOAuthClient, getApplicationPageList, getOAuthClientPageList, updateOAuthClient } from '@ark-iam/api'
import type { OAuthClientItem } from '@ark-iam/types'
import { useNavigate } from 'react-router-dom'
import { fmtTime, StatusTag, TypeTag } from '../../components/common'

interface AppOption {
  value: string
  label: string
}

export default function OAuthClientList() {
  const navigate = useNavigate()
  const [data, setData] = useState<OAuthClientItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<OAuthClientItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)
  const [appOptions, setAppOptions] = useState<AppOption[]>([])
  const [appLoading, setAppLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getOAuthClientPageList({ page, pageSize, name: keyword })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const loadApps = async () => {
    setAppLoading(true)
    try {
      const resp = await getApplicationPageList({ page: 1, pageSize: 100 })
      setAppOptions((resp?.list || []).map((a) => ({ value: a.appID, label: a.name })))
    } catch {
      /* 拦截器已提示 */
    } finally {
      setAppLoading(false)
    }
  }

  const handleCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
    void loadApps()
  }

  const handleEdit = (record: OAuthClientItem) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      type: record.type,
      status: record.status,
      tokenEndpointAuthMethod: record.tokenEndpointAuthMethod,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateOAuthClient({ applicationClientID: editing.applicationClientID, ...values })
        message.success('修改成功')
      } else {
        await createOAuthClient(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      void fetchData()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const columns: ColumnsType<OAuthClientItem> = [
    { title: 'ID', dataIndex: 'applicationClientID', key: 'applicationClientID', width: 80 },
    {
      title: '客户端ID',
      dataIndex: 'clientID',
      key: 'clientID',
      width: 240,
      render: (v: string) => <span style={{ fontFamily: 'monospace' }}>{v || '-'}</span>,
    },
    { title: '名称', dataIndex: 'name', key: 'name', ellipsis: true },
    { title: '所属应用ID', dataIndex: 'appID', key: 'appID', width: 110 },
    { title: '类型', dataIndex: 'type', key: 'type', width: 100, render: (v: string) => <TypeTag value={v} /> },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100, render: (v: string) => <StatusTag value={v} /> },
    { title: '创建时间', key: 'createdAt', width: 160, render: (_, r) => fmtTime(r.createdAt) },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => navigate(`/oauthClient/${r.applicationClientID}`)}>
            详情
          </Button>
          <Button type="link" size="small" onClick={() => handleEdit(r)}>
            编辑
          </Button>
          <Popconfirm
            title="确认删除该客户端？"
            onConfirm={async () => {
              try {
                await deleteOAuthClient(r.applicationClientID)
                message.success('删除成功')
                void fetchData()
              } catch {
                /* 拦截器已提示 */
              }
            }}
          >
            <Button type="link" size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title="OAuth 客户端"
      description="管理应用接入的 OIDC/OAuth 客户端与密钥"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建客户端
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按名称搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => { setKeyword(v); setPage(1) }}
        />
      </div>
      <Table<OAuthClientItem>
        rowKey="applicationClientID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1100 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />

      <Modal
        title={editing ? '编辑客户端' : '新建客户端'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          {!editing && (
            <Form.Item name="appID" label="所属应用" rules={[{ required: true, message: '请选择所属应用' }]}>
              <Select
                placeholder="选择所属应用"
                loading={appLoading}
                options={appOptions}
                showSearch
                optionFilterProp="label"
              />
            </Form.Item>
          )}
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="客户端名称" />
          </Form.Item>
          <Form.Item name="type" label="类型" initialValue="first_party" rules={[{ required: true, message: '请选择类型' }]}>
            <Select
              options={[
                { value: 'first_party', label: '第一方' },
                { value: 'third_party', label: '第三方' },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="tokenEndpointAuthMethod"
            label="令牌端点认证方式"
            initialValue="client_secret_basic"
            rules={[{ required: true, message: '请选择认证方式' }]}
          >
            <Select
              options={[
                { value: 'client_secret_basic', label: 'client_secret_basic' },
                { value: 'client_secret_post', label: 'client_secret_post' },
                { value: 'none', label: 'none' },
              ]}
            />
          </Form.Item>
          {editing && (
            <Form.Item name="status" label="状态" initialValue="enable" rules={[{ required: true, message: '请选择状态' }]}>
              <Select
                options={[
                  { value: 'enable', label: '启用' },
                  { value: 'disable', label: '停用' },
                ]}
              />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </PageContainer>
  )
}
