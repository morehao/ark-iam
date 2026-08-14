import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, message, Modal, Popconfirm, Select, Space, Switch, Table } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { PageContainer } from '@ark-iam/ui'
import { createTenant, deleteTenant, getTenantPageList, updateTenant } from '@ark-iam/api'
import type { TenantItem } from '@ark-iam/types'
import { fmtTime, SuspendedTag, TypeTag } from '../../components/common'

export default function TenantList() {
  const [data, setData] = useState<TenantItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTenantPageList({ page, pageSize, name: keyword })
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
    form.setFieldsValue({ type: 'customer' })
    setModalOpen(true)
  }

  const handleEdit = (record: TenantItem) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      type: record.type,
      tag: record.tag,
      code: record.code,
      dbUser: record.dbUser,
      isSuspended: record.isSuspended,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateTenant({ tenantID: editing.tenantID, ...values })
        message.success('修改成功')
      } else {
        await createTenant(values)
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

  const handleDelete = async (record: TenantItem) => {
    try {
      await deleteTenant(record.tenantID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<TenantItem> = [
    { title: 'ID', dataIndex: 'tenantID', key: 'tenantID', width: 70 },
    { title: '租户名', dataIndex: 'name', key: 'name', width: 180, ellipsis: true },
    {
      title: '编码',
      dataIndex: 'code',
      key: 'code',
      width: 150,
      render: (v: string) => <span style={{ fontFamily: 'monospace' }}>{v || '-'}</span>,
    },
    { title: '类型', dataIndex: 'type', key: 'type', width: 100, render: (v: string) => <TypeTag value={v} /> },
    { title: '标签', dataIndex: 'tag', key: 'tag', width: 140, render: (v: string) => v || '-' },
    {
      title: '状态',
      dataIndex: 'isSuspended',
      key: 'isSuspended',
      width: 100,
      render: (v: number) => <SuspendedTag value={v} />,
    },
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
          <Popconfirm title="确认删除该租户？" onConfirm={() => void handleDelete(r)}>
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
      title="租户管理"
      description="平台租户生命周期管理"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建租户
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按租户名搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
      </div>
      <Table<TenantItem>
        rowKey="tenantID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1000 }}
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
        title={editing ? '编辑租户' : '新建租户'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="租户名称" rules={[{ required: true, message: '请输入租户名称' }]}>
            <Input placeholder="租户名称" />
          </Form.Item>
          <Form.Item name="type" label="类型">
            <Select
              options={[
                { value: 'customer', label: '客户' },
                { value: 'platform', label: '平台' },
              ]}
            />
          </Form.Item>
          <Form.Item name="tag" label="标签">
            <Input placeholder="选填" />
          </Form.Item>
          <Form.Item name="code" label="编码" extra="留空自动生成">
            <Input placeholder="选填" />
          </Form.Item>
          <Form.Item name="dbUser" label="数据库用户">
            <Input placeholder="选填" />
          </Form.Item>
          {editing && (
            <Form.Item
              name="isSuspended"
              label="状态"
              valuePropName="checked"
              getValueFromEvent={(c: boolean) => (c ? 0 : 1)}
              getValueProps={(v?: number) => ({ checked: v !== 1 })}
              initialValue={0}
            >
              <Switch checkedChildren="正常" unCheckedChildren="挂起" />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </PageContainer>
  )
}
