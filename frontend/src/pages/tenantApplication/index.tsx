import { useEffect, useState } from 'react'
import { Table, Button, Space, Modal, Form, Input, InputNumber, Select, message, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getTenantApplicationPageList, createTenantApplication, updateTenantApplication, deleteTenantApplication, TenantApplication } from '../../api/tenantApplication'

const TenantApplicationList = () => {
  const [data, setData] = useState<TenantApplication[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = async () => {
    setLoading(true)
    try {
      const resp = await getTenantApplicationPageList({ page, pageSize, status: statusFilter })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch (error) {
      console.error('获取租户应用列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize, statusFilter])

  const handleAdd = () => {
    setEditingId(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = async (record: TenantApplication) => {
    setEditingId(record.tenantAppId)
    form.setFieldsValue(record)
    setModalVisible(true)
  }

  const handleDelete = (record: TenantApplication) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除租户应用订阅(ID: ${record.tenantAppId})吗？`,
      onOk: async () => {
        try {
          await deleteTenantApplication(record.tenantAppId)
          message.success('删除成功')
          fetchData()
        } catch {
          console.error('删除租户应用失败')
        }
      },
    })
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editingId) {
        await updateTenantApplication({ tenantAppId: editingId, ...values })
        message.success('修改成功')
      } else {
        await createTenantApplication(values)
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

  const columns: ColumnsType<TenantApplication> = [
    { title: 'ID', dataIndex: 'tenantAppId', key: 'tenantAppId', width: 80 },
    { title: '租户ID', dataIndex: 'tenantId', key: 'tenantId', width: 80 },
    { title: '应用ID', dataIndex: 'appId', key: 'appId', width: 80 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (val: string) => val === 'enable' ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>,
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
      <h1>租户应用管理</h1>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Space>
          <Select
            placeholder="状态筛选"
            style={{ width: 120 }}
            allowClear
            value={statusFilter}
            onChange={(val) => { setStatusFilter(val); setPage(1) }}
          >
            <Select.Option value="enable">启用</Select.Option>
            <Select.Option value="disable">停用</Select.Option>
          </Select>
        </Space>
        <Button type="primary" onClick={handleAdd}>新建租户应用</Button>
      </div>
      <Table
        columns={columns}
        dataSource={data}
        rowKey="tenantAppId"
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
        title={editingId ? '编辑租户应用' : '新建租户应用'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        confirmLoading={submitLoading}
      >
        <Form form={form} layout="vertical">
          {!editingId && (
            <Form.Item name="appId" label="应用ID" rules={[{ required: true, message: '请输入应用ID' }]}>
              <InputNumber style={{ width: '100%' }} />
            </Form.Item>
          )}
          <Form.Item name="status" label="状态" initialValue="enable">
            <Select>
              <Select.Option value="enable">启用</Select.Option>
              <Select.Option value="disable">停用</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="config" label="配置(JSON)">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default TenantApplicationList
