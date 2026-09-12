import { useCallback, useEffect, useState } from 'react'
import { Table, Button, Space, Input, Modal, Form, Select, message } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { actionColumn, idColumn, nameColumn, PageContainer, RemoteSelect, SourceTag, STATUS_COL_WIDTH, EnableTag, tableScrollX, TAG_COL_WIDTH, timeColumn, tokens } from '@ark-iam/ui'
import { createOAuthClient, deleteOAuthClient, getApplicationPageList, getOAuthClientPageList, updateOAuthClient } from '@ark-iam/api'
import type { OAuthClientItem } from '@ark-iam/types'
import { useNavigate } from 'react-router-dom'

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

  // 应用可增长，所属应用下拉走服务端搜索（RemoteSelect），不再一次性只取前 100 条
  const fetchAppOptions = useCallback(async (search: string) => {
    const resp = await getApplicationPageList({ page: 1, pageSize: 50, name: search || undefined })
    return (resp?.list || []).map((a) => ({ value: a.appID, label: a.name }))
  }, [])

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

  const handleCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: OAuthClientItem) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
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

  const handleDelete = async (record: OAuthClientItem) => {
    try {
      await deleteOAuthClient(record.applicationClientID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<OAuthClientItem> = [
    idColumn<OAuthClientItem>({ dataIndex: 'applicationClientID' }),
    idColumn<OAuthClientItem>({ dataIndex: 'clientID', title: '客户端ID' }),
    nameColumn<OAuthClientItem>({
      title: '名称',
      dataIndex: 'name',
      onClick: (r) => navigate(`/oauthClient/${r.applicationClientID}`),
    }),
    idColumn<OAuthClientItem>({ dataIndex: 'appID', title: '所属应用ID' }),
    { title: '来源', dataIndex: 'source', key: 'source', width: TAG_COL_WIDTH, render: (v: string) => <SourceTag value={v} /> },
    { title: '状态', dataIndex: 'status', key: 'status', width: STATUS_COL_WIDTH, render: (v: string) => <EnableTag value={v} /> },
    timeColumn<OAuthClientItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<OAuthClientItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<OAuthClientItem>({
      max: 2,
      actions: (r) => [
        { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
        { key: 'delete', label: '删除', danger: true, confirm: '确认删除该客户端？', onClick: () => void handleDelete(r) },
      ],
    }),
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
        tableLayout="fixed"
        scroll={tableScrollX(columns)}
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
              <RemoteSelect placeholder="选择所属应用（输入名称搜索）" fetchOptions={fetchAppOptions} />
            </Form.Item>
          )}
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
            extra={editing?.source === 'builtin' ? '内置客户端的名称由平台版本定义，不可修改' : undefined}
          >
            <Input placeholder="客户端名称" disabled={editing?.source === 'builtin'} />
          </Form.Item>
          {editing && (
            <Form.Item label="来源">
              <SourceTag value={editing.source} />
              <span style={{ marginLeft: 8, color: tokens.textSecondary }}>来源不可修改</span>
            </Form.Item>
          )}
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
