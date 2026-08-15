import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, InputNumber, Modal, Popconfirm, Space, Table, message } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { EllipsisCell, IDCell, PageContainer } from '@ark-iam/ui'
import { createScope, deleteScope, getScopeDetail, getScopePageList, updateScope } from '@ark-iam/api'
import type { ScopeItem } from '@ark-iam/types'
import { fmtTime } from '../../components/common'

export default function ScopeList() {
  const [data, setData] = useState<ScopeItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<ScopeItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getScopePageList({ page, pageSize, name: keyword })
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

  const handleEdit = (record: ScopeItem) => {
    setEditing(record)
    form.setFieldsValue({ name: record.name, resourceID: record.resourceID, description: record.description })
    setModalOpen(true)
    getScopeDetail(record.scopeID)
      .then((detail) => {
        form.setFieldsValue({ name: detail.name, resourceID: detail.resourceID, description: detail.description })
      })
      .catch(() => {
        /* 详情拉取失败，回退行数据 */
      })
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateScope({ scopeID: editing.scopeID, ...values })
        message.success('修改成功')
      } else {
        await createScope(values)
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

  const columns: ColumnsType<ScopeItem> = [
    { title: 'ID', dataIndex: 'scopeID', key: 'scopeID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '权限名', dataIndex: 'name', key: 'name', width: 180 },
    { title: '资源ID', dataIndex: 'resourceID', key: 'resourceID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '描述', dataIndex: 'description', key: 'description', render: (v: string) => <EllipsisCell value={v} /> },
    {
      title: '创建时间',
      key: 'createdAt',
      width: 160,
      render: (_, r) => fmtTime(r.createdAt),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => handleEdit(r)}>
            编辑
          </Button>
          <Popconfirm
            title="确认删除该权限域？"
            onConfirm={async () => {
              try {
                await deleteScope(r.scopeID)
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
      title="权限域管理"
      description="管理 IAM 权限域及其关联资源"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建权限域
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按权限名搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => { setKeyword(v); setPage(1) }}
        />
      </div>
      <Table<ScopeItem>
        rowKey="scopeID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 800 }}
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
        title={editing ? '编辑权限域' : '新建权限域'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="权限名" rules={[{ required: true, message: '请输入权限名' }]}>
            <Input placeholder="如：只读访问" />
          </Form.Item>
          <Form.Item name="resourceID" label="资源ID" rules={[{ required: true, message: '请输入资源ID' }]}>
            <InputNumber style={{ width: '100%' }} placeholder="关联的资源 ID" min={1} precision={0} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea placeholder="选填" rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
