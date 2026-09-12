import { useCallback, useEffect, useState } from 'react'
import { Table, Button, Space, Input, Modal, Form, Select, message } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { actionColumn, CODE_COL_WIDTH, idColumn, NAME_COL_WIDTH, nameColumn, PageContainer, RemoteSelect, SourceTag, STATUS_COL_WIDTH, EnableTag, tableScrollX, TAG_COL_WIDTH, textColumn, timeColumn, tokens } from '@ark-iam/ui'
import { createOAuthClient, deleteOAuthClient, getApplicationPageList, getOAuthClientPageList, updateOAuthClient } from '@ark-iam/api'
import type { OAuthClientItem } from '@ark-iam/types'
import { useNavigate } from 'react-router-dom'
import { oauthClientDetailPath } from '../../routes'

// 客户端编码（= OIDC client_id）规则，与后端 model.ClientCodePattern 同口径：
// 小写字母开头，仅含小写字母与下划线（禁数字与连字符）。前端与后端各校验一份，改一处必须同步另一处。
const CLIENT_CODE_PATTERN = /^[a-z][a-z_]*$/

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
  // 内置控制台客户端：编码（= client_id）是网关 aud 白名单与前端构建期默认值的取值来源，保持只读
  const builtinClient = editing?.source === 'builtin'

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
      code: record.code,
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
    nameColumn<OAuthClientItem>({
      title: '名称',
      dataIndex: 'name',
      // 名称即详情入口（列规范 R2）；路由 path 取自 routes.ts，与 App.tsx 注册保持一致
      onClick: (r) => navigate(oauthClientDetailPath(r.applicationClientID)),
    }),
    // 列名与后端字段名同构（DTO/DB 都是 code）：客户端编码 = OIDC client_id
    textColumn<OAuthClientItem>({ title: '客户端编码', dataIndex: 'code', width: CODE_COL_WIDTH, monospace: true }),
    textColumn<OAuthClientItem>({ title: '所属应用', dataIndex: 'appName', width: NAME_COL_WIDTH }),
    { title: '来源', dataIndex: 'source', key: 'source', width: TAG_COL_WIDTH, render: (v: string) => <SourceTag value={v} /> },
    { title: '状态', dataIndex: 'status', key: 'status', width: STATUS_COL_WIDTH, render: (v: string) => <EnableTag value={v} /> },
    timeColumn<OAuthClientItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<OAuthClientItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<OAuthClientItem>({
      max: 2,
      actions: (r) => [
        { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
        {
          key: 'delete',
          label: '删除',
          danger: true,
          // 内置客户端（source=builtin，网关 aud 白名单与前端构建期 client_id 的来源）禁删：置灰保留展示；
          // 后端 svcapplicationclient.Delete 以 ApplicationClientBuiltInErr 兜底。
          disabled: r.source === 'builtin',
          confirm: '确认删除该客户端？',
          onClick: () => void handleDelete(r),
        },
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
            name="code"
            label="客户端编码"
            tooltip={
              builtinClient
                ? '内置控制台的 OIDC 身份，由平台版本定义，不可修改'
                : '即 OIDC 的 client_id。改动后使用它的应用需同步自己的 client_id 配置，且其已签发令牌会失效'
            }
            rules={[
              { required: true, message: '请输入客户端编码' },
              { pattern: CLIENT_CODE_PATTERN, message: '以小写字母开头，仅含小写字母与下划线' },
            ]}
          >
            <Input placeholder="唯一编码，如 iam_client" disabled={builtinClient} />
          </Form.Item>
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
          >
            <Input placeholder="客户端名称" />
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
