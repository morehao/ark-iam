import { useCallback, useEffect, useState } from 'react'
import { Button, Form, message, Modal, Select, Space, Table } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { actionColumn, idColumn, NAME_COL_WIDTH, PageContainer, RemoteSelect, SourceTag, STATUS_COL_WIDTH, EnableTag, tableScrollX, TAG_COL_WIDTH, textColumn, timeColumn } from '@ark-iam/ui'
import {
  createTenantApplication,
  deleteTenantApplication,
  getApplicationPageList,
  getTenantApplicationPageList,
  getTenantPageList,
  updateTenantApplication,
} from '@ark-iam/api'
import type { TenantApplicationItem } from '@ark-iam/types'

export default function TenantApplicationList() {
  const [data, setData] = useState<TenantApplicationItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [tenantFilter, setTenantFilter] = useState<string | undefined>(undefined)
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantApplicationItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  // 租户/应用主键都是 UUID 字符串（只能选择不能手输），且两者都会增长：
  // 下拉一律走 RemoteSelect 服务端搜索，不再一次性只取前 100 条。
  const fetchTenantOptions = useCallback(async (keyword: string) => {
    const resp = await getTenantPageList({ page: 1, pageSize: 50, name: keyword || undefined })
    return (resp?.list || []).map((t) => ({ value: t.tenantID, label: t.name }))
  }, [])

  const fetchAppOptions = useCallback(async (keyword: string) => {
    const resp = await getApplicationPageList({ page: 1, pageSize: 50, name: keyword || undefined })
    return (resp?.list || []).map((a) => ({ value: a.appID, label: a.name }))
  }, [])

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTenantApplicationPageList({
        page,
        pageSize,
        tenantID: tenantFilter,
        status: statusFilter,
      })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, tenantFilter, statusFilter])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ status: 'enable' })
    setModalOpen(true)
  }

  const handleEdit = (record: TenantApplicationItem) => {
    setEditing(record)
    form.setFieldsValue({
      tenantID: record.tenantID,
      appID: record.appID,
      status: record.status,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        const { tenantID: _tenantID, appID: _appID, ...rest } = values
        await updateTenantApplication({ tenantAppID: editing.tenantAppID, ...rest })
        message.success('修改成功')
      } else {
        await createTenantApplication(values)
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

  const handleDelete = async (record: TenantApplicationItem) => {
    try {
      await deleteTenantApplication(record.tenantAppID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<TenantApplicationItem> = [
    idColumn<TenantApplicationItem>({ dataIndex: 'tenantAppID' }),
    textColumn<TenantApplicationItem>({ title: '租户', dataIndex: 'tenantName', width: NAME_COL_WIDTH }),
    textColumn<TenantApplicationItem>({ title: '应用', dataIndex: 'appName', width: NAME_COL_WIDTH }),
    // 应用来源（后端回填所属应用 source）：builtin=内置应用，其订阅由系统开通、不可删除，
    // 展示出来才能解释该行为何没有「删除」。
    { title: '应用来源', dataIndex: 'appSource', key: 'appSource', width: TAG_COL_WIDTH, render: (v: string) => <SourceTag value={v} /> },
    { title: '状态', dataIndex: 'status', key: 'status', width: STATUS_COL_WIDTH, render: (v: string) => <EnableTag value={v} /> },
    timeColumn<TenantApplicationItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<TenantApplicationItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<TenantApplicationItem>({
      max: 2,
      actions: (r) => [
        { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
        {
          key: 'delete',
          label: '删除',
          danger: true,
          // 订阅的是内置应用（source=builtin，平台管理后台/租户管理后台）即禁删：它由种子或
          // ProvisionTenantAdmin 系统开通，删除会让对应控制台失去菜单；置灰保留展示，下线请改用「停用」。
          // 后端 svctenantapplication.Delete 以 TenantApplicationBuiltInErr 兜底。
          disabled: r.appSource === 'builtin',
          confirm: '确认删除该订阅？',
          onClick: () => void handleDelete(r),
        },
      ],
    }),
  ]

  return (
    <PageContainer
      title="租户应用"
      description="租户对应用的订阅关系（平台侧跨租户开通，归属租户按选择指定）"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建订阅
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16, display: 'flex', gap: 12 }}>
        <RemoteSelect
          allowClear
          width={220}
          placeholder="租户筛选（输入名称搜索）"
          value={tenantFilter}
          onChange={(v) => {
            setTenantFilter(v)
            setPage(1)
          }}
          fetchOptions={fetchTenantOptions}
        />
        <Select
          allowClear
          placeholder="状态筛选"
          style={{ width: 140 }}
          value={statusFilter}
          onChange={(v) => {
            setStatusFilter(v)
            setPage(1)
          }}
          options={[
            { value: 'enable', label: '启用' },
            { value: 'disable', label: '停用' },
          ]}
        />
      </div>
      <Table<TenantApplicationItem>
        rowKey="tenantAppID"
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
        title={editing ? '编辑订阅' : '新建订阅'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="tenantID" label="租户" rules={[{ required: true, message: '请选择租户' }]}>
            {/* 编辑态归属不可改（更新接口不含 tenantID/appID），下拉只用于回显名称 */}
            <RemoteSelect
              placeholder="选择租户（输入名称搜索）"
              fetchOptions={fetchTenantOptions}
              disabled={!!editing}
              initialLabel={editing?.tenantName}
            />
          </Form.Item>
          <Form.Item name="appID" label="应用" rules={[{ required: true, message: '请选择应用' }]}>
            <RemoteSelect
              placeholder="选择应用（输入名称搜索）"
              fetchOptions={fetchAppOptions}
              disabled={!!editing}
              initialLabel={editing?.appName}
            />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              options={[
                { value: 'enable', label: '启用' },
                { value: 'disable', label: '停用' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
