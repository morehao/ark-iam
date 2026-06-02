import { useEffect, useState } from 'react'
import { Table, Button, Space, Input, Modal, Form, Select, message, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getTenantPageList, createTenant, updateTenant, deleteTenant, getTenantDetail, Tenant } from '../../api/tenant'

const TenantList = () => {
  const [data, setData] = useState<Tenant[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [modalVisible, setModalVisible] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = async () => {
    setLoading(true)
    try {
      const resp = await getTenantPageList({ page, pageSize, keyword })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch (error) {
      console.error('获取租户列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize, keyword])

  const handleAdd = () => {
    setEditingId(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = async (record: Tenant) => {
    setEditingId(record.tenantID)
    try {
      const detail = await getTenantDetail(record.tenantID)
      form.setFieldsValue(detail)
    } catch {
      form.setFieldsValue(record)
    }
    setModalVisible(true)
  }

  const handleDelete = (record: Tenant) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除租户"${record.name}"吗？`,
      onOk: async () => {
        try {
          await deleteTenant(record.tenantID)
          message.success('删除成功')
          fetchData()
        } catch {
          console.error('删除租户失败')
        }
      },
    })
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editingId) {
        await updateTenant({ tenantID: editingId, ...values })
        message.success('修改成功')
      } else {
        await createTenant(values)
        message.success('创建成功')
      }
      setModalVisible(false)
      fetchData()
    } catch (error) {
      if (error instanceof Error) {
        console.error('提交失败:', error.message)
      }
    } finally {
      setSubmitLoading(false)
    }
  }

  const columns: ColumnsType<Tenant> = [
    { title: 'ID', dataIndex: 'tenantID', key: 'tenantID', width: 80 },
    { title: '租户名称', dataIndex: 'name', key: 'name' },
    { title: '标签', dataIndex: 'tag', key: 'tag' },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (val: string) => val === 'platform' ? <Tag color="blue">平台</Tag> : <Tag>客户</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'isSuspended',
      key: 'isSuspended',
      render: (val: number) => val === 1 ? <Tag color="red">挂起</Tag> : <Tag color="green">正常</Tag>,
    },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => handleEdit(record)}>编辑</Button>
          <Button type="link" danger onClick={() => handleDelete(record)}>删除</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <h1>租户管理</h1>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Input.Search
          placeholder="搜索租户名称"
          style={{ width: 200 }}
          onSearch={(value) => setKeyword(value)}
        />
        <Button type="primary" onClick={handleAdd}>新建租户</Button>
      </div>
      <Table
        columns={columns}
        dataSource={data}
        rowKey="tenantID"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />
      <Modal
        title={editingId ? '编辑租户' : '新建租户'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        confirmLoading={submitLoading}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="租户名称" rules={[{ required: true, message: '请输入租户名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="tag" label="标签">
            <Input />
          </Form.Item>
          <Form.Item name="type" label="类型" initialValue="customer">
            <Select>
              <Select.Option value="customer">客户租户</Select.Option>
              <Select.Option value="platform">平台租户</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default TenantList
