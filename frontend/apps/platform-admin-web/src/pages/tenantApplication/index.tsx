import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, InputNumber, message, Modal, Popconfirm, Select, Space, Table } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { fmtTime, IDCell, PageContainer, StatusTag } from '@ark-iam/ui'
import {
  createTenantApplication,
  deleteTenantApplication,
  getTenantApplicationPageList,
  updateTenantApplication,
} from '@ark-iam/api'
import type { TenantApplicationItem } from '@ark-iam/types'

export default function TenantApplicationList() {
  const [data, setData] = useState<TenantApplicationItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantApplicationItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTenantApplicationPageList({ page, pageSize, status: statusFilter })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, statusFilter])

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
      appID: record.appID,
      status: record.status,
      config: record.config,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        const { appID: _appId, ...rest } = values
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
    { title: 'ID', dataIndex: 'tenantAppID', key: 'tenantAppID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '租户ID', dataIndex: 'tenantID', key: 'tenantID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '应用ID', dataIndex: 'appID', key: 'appID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100, render: (v: string) => <StatusTag value={v} /> },
    { title: '创建时间', key: 'createdAt', width: 170, render: (_, r) => fmtTime(r.createdAt) },
    {
      title: '操作',
      key: 'action',
      width: 140,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => handleEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="确认删除该订阅？" onConfirm={() => void handleDelete(r)}>
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
      title="租户应用"
      description="租户对应用的订阅关系"
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
      <div style={{ marginBottom: 16 }}>
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
        scroll={{ x: 700 }}
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
          <Form.Item name="appID" label="应用ID" rules={[{ required: true, message: '请输入应用ID' }]}>
            <InputNumber style={{ width: '100%' }} placeholder="应用ID" disabled={!!editing} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              options={[
                { value: 'enable', label: '启用' },
                { value: 'disable', label: '停用' },
              ]}
            />
          </Form.Item>
          <Form.Item name="config" label="配置（JSON）">
            <Input.TextArea rows={4} placeholder='选填，如 {"region":"cn"}' />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
