import { useEffect, useState } from 'react'
import { Table, Button, Space, Input, Modal, Form, Select, message, Tag } from 'antd'
import { useNavigate } from 'react-router-dom'
import type { ColumnsType } from 'antd/es/table'
import { getOAuthClientPageList, createOAuthClient, updateOAuthClient, deleteOAuthClient, getOAuthClientDetail, OAuthClient } from '../../api/oauthClient'

const OAuthClientList = () => {
  const navigate = useNavigate()
  const [data, setData] = useState<OAuthClient[]>([])
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
      const resp = await getOAuthClientPageList({ page, pageSize, name: keyword })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch (error) {
      console.error('获取OAuth客户端列表失败:', error)
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

  const handleEdit = async (record: OAuthClient) => {
    setEditingId(record.oauthClientId)
    try {
      const detail = await getOAuthClientDetail(record.oauthClientId)
      form.setFieldsValue(detail)
    } catch {
      form.setFieldsValue(record)
    }
    setModalVisible(true)
  }

  const handleDelete = (record: OAuthClient) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除OAuth客户端"${record.name}"吗？`,
      onOk: async () => {
        try {
          await deleteOAuthClient(record.oauthClientId)
          message.success('删除成功')
          fetchData()
        } catch {
          console.error('删除OAuth客户端失败')
        }
      },
    })
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editingId) {
        await updateOAuthClient({ oauthClientId: editingId, ...values })
        message.success('修改成功')
      } else {
        await createOAuthClient(values)
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

  const columns: ColumnsType<OAuthClient> = [
    { title: 'ID', dataIndex: 'oauthClientId', key: 'oauthClientId', width: 80 },
    { title: '客户端ID', dataIndex: 'clientID', key: 'clientID' },
    { title: '名称', dataIndex: 'name', key: 'name' },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (val: string) => val === 'third_party' ? <Tag color="orange">第三方</Tag> : <Tag color="blue">第一方</Tag>,
    },
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
          <Button type="link" onClick={() => navigate(`/oauthClient/${record.oauthClientId}`)}>详情</Button>
          <Button type="link" onClick={() => handleEdit(record)}>编辑</Button>
          <Button type="link" danger onClick={() => handleDelete(record)}>删除</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <h1>OAuth 客户端管理</h1>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Input.Search
          placeholder="搜索客户端名称"
          style={{ width: 200 }}
          onSearch={(value) => setKeyword(value)}
        />
        <Button type="primary" onClick={handleAdd}>新建客户端</Button>
      </div>
      <Table
        columns={columns}
        dataSource={data}
        rowKey="oauthClientId"
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
        title={editingId ? '编辑OAuth客户端' : '新建OAuth客户端'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        confirmLoading={submitLoading}
        width={640}
      >
        <Form form={form} layout="vertical">
          {!editingId && (
            <Form.Item name="appId" label="所属应用ID" rules={[{ required: true, message: '请输入应用ID' }]}>
              <Input type="number" />
            </Form.Item>
          )}
          <Form.Item name="name" label="客户端名称" rules={[{ required: true, message: '请输入客户端名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="客户端类型" initialValue="first_party">
            <Select>
              <Select.Option value="first_party">第一方</Select.Option>
              <Select.Option value="third_party">第三方</Select.Option>
            </Select>
          </Form.Item>
          {editingId && (
            <Form.Item name="status" label="状态" initialValue="enable">
              <Select>
                <Select.Option value="enable">启用</Select.Option>
                <Select.Option value="disable">停用</Select.Option>
              </Select>
            </Form.Item>
          )}
          <Form.Item name="tokenEndpointAuthMethod" label="令牌端点认证方式" initialValue="client_secret_basic">
            <Select>
              <Select.Option value="client_secret_basic">client_secret_basic</Select.Option>
              <Select.Option value="client_secret_post">client_secret_post</Select.Option>
              <Select.Option value="none">none</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default OAuthClientList
