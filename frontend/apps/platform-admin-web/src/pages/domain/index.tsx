import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, Modal, Select, Space, Table, message } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { actionColumn, CODE_COL_WIDTH, idColumn, PageContainer, STATUS_COL_WIDTH, tableScrollX, textColumn, timeColumn, VerifiedTag } from '@ark-iam/ui'
import { createDomain, deleteDomain, getDomainDetail, getDomainPageList, updateDomain } from '@ark-iam/api'
import type { DomainItem, DomainVerificationStatus } from '@ark-iam/types'

export default function DomainList() {
  const [data, setData] = useState<DomainItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<DomainItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getDomainPageList({ page, pageSize, domain: keyword })
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

  const handleEdit = (record: DomainItem) => {
    setEditing(record)
    form.setFieldsValue({ domain: record.domain, verificationStatus: record.verificationStatus })
    setModalOpen(true)
    getDomainDetail(record.id)
      .then((detail) => {
        form.setFieldsValue({ domain: detail.domain, verificationStatus: detail.verificationStatus })
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
        await updateDomain({ id: editing.id, domain: values.domain, verificationStatus: values.verificationStatus })
        message.success('修改成功')
      } else {
        await createDomain(values.domain)
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

  const handleDelete = async (record: DomainItem) => {
    try {
      await deleteDomain(record.id)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<DomainItem> = [
    idColumn<DomainItem>({ dataIndex: 'id' }),
    textColumn<DomainItem>({ title: '域名', dataIndex: 'domain', width: CODE_COL_WIDTH, monospace: true }),
    {
      title: '验证状态',
      dataIndex: 'verificationStatus',
      key: 'verificationStatus',
      width: STATUS_COL_WIDTH,
      render: (v: DomainVerificationStatus) => <VerifiedTag value={v} />,
    },
    timeColumn<DomainItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<DomainItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<DomainItem>({
      max: 2,
      actions: (r) => [
        { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
        { key: 'delete', label: '删除', danger: true, confirm: '确认删除该域名？', onClick: () => void handleDelete(r) },
      ],
    }),
  ]

  return (
    <PageContainer
      title="域名管理"
      description="管理平台域名及其验证状态"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建域名
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按域名搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => { setKeyword(v); setPage(1) }}
        />
      </div>
      <Table<DomainItem>
        rowKey="id"
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
        title={editing ? '编辑域名' : '新建域名'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={480}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="domain" label="域名" rules={[{ required: true, message: '请输入域名' }]}>
            <Input placeholder="如：example.com" />
          </Form.Item>
          {editing && (
            <Form.Item name="verificationStatus" label="验证状态" rules={[{ required: true, message: '请选择验证状态' }]}>
              <Select
                options={[
                  { value: 'unverified', label: '未验证' },
                  { value: 'verified', label: '已验证' },
                ]}
              />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </PageContainer>
  )
}
