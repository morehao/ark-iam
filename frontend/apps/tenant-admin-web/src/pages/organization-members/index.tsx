import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Avatar,
  Button,
  Card,
  Drawer,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  TreeSelect,
  message,
} from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { fmtTime, PageContainer, SuspendedTag, tokens } from '@ark-iam/ui'
import type { MemberItem, OrganizationItem, UserOrganizationItem } from '@ark-iam/types'
import { getOrganizationTree } from '../../api/organization'
import { createTenantUser, getTenantMemberPageList, getTenantUserDetail, updateTenantUser } from '../../api/user'

type OrgOf = { id: string; name: string }

export default function OrganizationMembersPage() {
  const [orgTree, setOrgTree] = useState<OrganizationItem[]>([])
  const [orgLoading, setOrgLoading] = useState(false)

  const [data, setData] = useState<MemberItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)
  const [filterForm] = Form.useForm()
  const [query, setQuery] = useState<{ organizationID?: string; keyword?: string; isSuspended?: boolean }>({})

  // 新增 / 编辑成员
  const [editorOpen, setEditorOpen] = useState(false)
  const [editing, setEditing] = useState<MemberItem | null>(null)
  const [editorForm] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)

  // 成员详情
  const [detail, setDetail] = useState<MemberItem | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)

  const loadTree = useCallback(async () => {
    setOrgLoading(true)
    try {
      const resp = await getOrganizationTree()
      setOrgTree(resp.list || [])
    } finally {
      setOrgLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadTree()
  }, [loadTree])

  const loadMembers = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTenantMemberPageList({
        page,
        pageSize,
        keyword: query.keyword || undefined,
        isSuspended: query.isSuspended,
        organizationID: query.organizationID || undefined,
      })
      setData(resp.list || [])
      setTotal(resp.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, query])

  useEffect(() => {
    void loadMembers()
  }, [loadMembers])

  const treeSelectData = useMemo(() => toTreeSelect(orgTree), [orgTree])

  const openCreate = () => {
    setEditing(null)
    editorForm.resetFields()
    editorForm.setFieldsValue({ secondaryOrgIDs: [], leaderOrgIDs: [] })
    setEditorOpen(true)
  }

  const openEdit = (record: MemberItem) => {
    setEditing(record)
    const prim = pickOrgs(record.organizations, 'primary')[0]
    const secondaries = pickOrgs(record.organizations, 'secondary').map((o) => o.id)
    const leaders = pickOrgs(record.organizations, 'leader').map((o) => o.id)
    editorForm.setFieldsValue({
      name: record.name,
      username: record.username,
      primaryEmail: record.primaryEmail,
      primaryPhone: record.primaryPhone,
      primaryOrgID: prim?.id,
      secondaryOrgIDs: secondaries,
      leaderOrgIDs: leaders,
      isSuspended: record.isSuspended,
    })
    setEditorOpen(true)
  }

  const submitEditor = async () => {
    try {
      const values = await editorForm.validateFields()
      setSubmitting(true)
      if (editing) {
        // 编辑：全量提交联系方式与主/参与/负责部门（若无则传空数组清空该维度；主部门必填）
        await updateTenantUser({
          userID: editing.userID,
          username: values.username || '',
          primaryEmail: values.primaryEmail || '',
          primaryPhone: values.primaryPhone || '',
          primaryOrgID: values.primaryOrgID,
          secondaryOrgIDs: values.secondaryOrgIDs || [],
          leaderOrgIDs: values.leaderOrgIDs || [],
          isSuspended: values.isSuspended,
        })
        message.success('保存成功')
      } else {
        await createTenantUser({
          name: values.name,
          username: values.username,
          primaryEmail: values.primaryEmail,
          primaryPhone: values.primaryPhone,
          password: values.password,
          isSuspended: values.isSuspended,
          organizationIDs: [values.primaryOrgID],
          secondaryOrgIDs: values.secondaryOrgIDs || [],
          leaderOrgIDs: values.leaderOrgIDs || [],
        })
        message.success('创建成功')
      }
      setEditorOpen(false)
      setPage(1)
      void loadTree()
      void loadMembers()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitting(false)
    }
  }

  const openDetail = async (record: MemberItem) => {
    setDetail(record)
    setDetailOpen(true)
    try {
      const d = await getTenantUserDetail(record.userID)
      setDetail({ ...record, organizations: d.organizations || [] })
    } catch {
      /* 拦截器已提示 */
    }
  }

  const memberColumns: ColumnsType<MemberItem> = [
    {
      title: '用户',
      key: 'user',
      width: 200,
      render: (_, r) => (
        <Space>
          <Avatar size={28}>{r.name?.charAt(0)?.toUpperCase() || 'U'}</Avatar>
          <Space direction="vertical" size={0}>
            <span style={{ fontWeight: 500 }}>{r.name || '-'}</span>
            <span style={{ fontSize: 12, color: tokens.textPlaceholder }}>@{r.username || '-'}</span>
          </Space>
        </Space>
      ),
    },
    { title: '邮箱', dataIndex: 'primaryEmail', key: 'primaryEmail', render: (v: string) => v || '-' },
    { title: '手机号', dataIndex: 'primaryPhone', key: 'primaryPhone', width: 130, render: (v: string) => v || '-' },
    {
      title: '状态',
      dataIndex: 'isSuspended',
      key: 'isSuspended',
      width: 90,
      render: (v: boolean) => <SuspendedTag value={v} />,
    },
    {
      title: '主部门',
      dataIndex: 'organizations',
      key: 'primaryOrg',
      width: 140,
      render: (orgs: UserOrganizationItem[]) => {
        const prim = pickOrgs(orgs, 'primary')[0]
        return prim ? <Tag color="gold">{prim.name || '-'}</Tag> : <Tag>未设置</Tag>
      },
    },
    { title: '角色数', dataIndex: 'roleCount', key: 'roleCount', width: 80, render: (v: number) => v || 0 },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 160, render: (v?: number) => fmtTime(v) },
    {
      title: '操作',
      key: 'action',
      width: 140,
      render: (_, r) => (
        <Space size={0}>
          <Button type="link" size="small" onClick={() => void openDetail(r)}>
            详情
          </Button>
          <Button type="link" size="small" onClick={() => openEdit(r)}>
            编辑
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title="成员管理"
      description="以人为维度管理租户内全部成员，维护主部门/参与部门/负责部门"
      extra={<Button icon={<ReloadOutlined />} onClick={() => void loadMembers()}>刷新</Button>}
    >
      <Card style={{ borderRadius: 12 }} styles={{ body: { padding: '16px 24px 24px' } }}>
        <Form
          form={filterForm}
          layout="inline"
          style={{ marginBottom: 16, rowGap: 12 }}
          onFinish={(v: { organizationID?: string; keyword?: string; isSuspended?: boolean }) => {
            setQuery({
              organizationID: v.organizationID,
              keyword: v.keyword,
              isSuspended: v.isSuspended,
            })
            setPage(1)
          }}
        >
          <Form.Item name="organizationID" label="部门">
            <TreeSelect
              treeData={treeSelectData}
              treeDefaultExpandAll
              allowClear
              placeholder="请选择部门"
              style={{ width: 220 }}
              loading={orgLoading}
            />
          </Form.Item>
          <Form.Item name="isSuspended" label="状态">
            <Select
              allowClear
              placeholder="请选择状态"
              style={{ width: 120 }}
              options={[
                { label: '正常', value: false },
                { label: '挂起', value: true },
              ]}
            />
          </Form.Item>
          <Form.Item name="keyword" label="关键词">
            <Input placeholder="姓名/用户名/邮箱/手机" allowClear prefix={<SearchOutlined />} style={{ width: 220 }} />
          </Form.Item>
          <Form.Item style={{ marginLeft: 'auto', marginRight: 0 }}>
            <Space>
              <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                查询
              </Button>
              <Button
                onClick={() => {
                  filterForm.resetFields()
                  setQuery({})
                  setPage(1)
                }}
              >
                重置
              </Button>
            </Space>
          </Form.Item>
        </Form>

        <div style={{ borderBottom: `1px solid ${tokens.border}`, marginBottom: 16 }} />

        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新增成员
          </Button>
        </div>

        <div style={{ borderBottom: `1px solid ${tokens.border}`, marginBottom: 16 }} />

        <Table<MemberItem>
          rowKey={(r) => r.userID}
          loading={loading}
          columns={memberColumns}
          dataSource={data}
          scroll={{ x: 1000 }}
          locale={{ emptyText: '暂无成员' }}
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
      </Card>

      {/* 新增 / 编辑成员 */}
      <Modal
        title={editing ? '编辑成员' : '新增成员'}
        open={editorOpen}
        onOk={() => void submitEditor()}
        onCancel={() => setEditorOpen(false)}
        confirmLoading={submitting}
        destroyOnClose
        width={620}
      >
        <Form form={editorForm} layout="vertical">
          {!editing && (
            <Form.Item name="name" label="姓名" rules={[{ required: true, message: '请输入姓名' }]}>
              <Input placeholder="如：张三" />
            </Form.Item>
          )}
          <Form.Item name="primaryOrgID" label="主部门" rules={[{ required: true, message: '请选择主部门' }]}>
            <TreeSelect
              treeData={treeSelectData}
              treeDefaultExpandAll
              placeholder="选择主部门（行政归属，唯一）"
            />
          </Form.Item>
          <Form.Item name="secondaryOrgIDs" label="参与部门">
            <TreeSelect
              treeData={treeSelectData}
              treeDefaultExpandAll
              multiple
              allowClear
              placeholder="选择参与部门（可多个，跨部门协作）"
            />
          </Form.Item>
          <Form.Item name="leaderOrgIDs" label="负责部门">
            <TreeSelect
              treeData={treeSelectData}
              treeDefaultExpandAll
              multiple
              allowClear
              placeholder="选择负责部门（可多个，但每部门至多一位负责人）"
            />
          </Form.Item>
          <Form.Item
            name="primaryEmail"
            label="邮箱"
            dependencies={['primaryPhone']}
            rules={[
              {
                validator: (_, v) => {
                  const phone = editorForm.getFieldValue('primaryPhone')
                  if ((!v || v === '') && (!phone || phone === '')) {
                    return Promise.reject(new Error('邮箱和手机号至少填写一个'))
                  }
                  return Promise.resolve()
                },
              },
            ]}
          >
            <Input placeholder="邮箱" />
          </Form.Item>
          <Form.Item name="primaryPhone" label="手机号" dependencies={['primaryEmail']}>
            <Input placeholder="手机号" />
          </Form.Item>
          <Form.Item name="username" label="用户名">
            <Input placeholder="可空，全局用户名" />
          </Form.Item>
          {!editing && (
            <Form.Item name="password" label="初始密码">
              <Input.Password placeholder="提供后该用户可登录" />
            </Form.Item>
          )}
          <Form.Item name="isSuspended" label="状态" valuePropName="checked" initialValue={false}>
            <Switch checkedChildren="挂起" unCheckedChildren="正常" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 成员详情 */}
      <Drawer title="成员详情" width={520} open={detailOpen} onClose={() => setDetailOpen(false)}>
        {detail && (
          <Tabs
            items={[
              {
                key: 'info',
                label: '基础信息',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Space>
                      <Avatar size={48}>{detail.name?.charAt(0)?.toUpperCase() || 'U'}</Avatar>
                      <Space direction="vertical" size={0}>
                        <span style={{ fontWeight: 600, fontSize: 16 }}>{detail.name || '-'}</span>
                        <span style={{ color: tokens.textPlaceholder }}>@{detail.username || '-'}</span>
                      </Space>
                    </Space>
                    <div>
                      <Tag>{detail.isSuspended ? '挂起' : '正常'}</Tag>
                      <span style={{ marginLeft: 8 }}>用户ID：{detail.userID}</span>
                    </div>
                    <div>邮箱：{detail.primaryEmail || '-'}</div>
                    <div>手机号：{detail.primaryPhone || '-'}</div>
                    <div>角色数：{detail.roleCount || 0}</div>
                    <div>创建时间：{fmtTime(detail.createdAt)}</div>
                  </Space>
                ),
              },
              {
                key: 'org',
                label: '组织关系',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <OrgRow label="主部门" color="gold" orgs={pickOrgs(detail.organizations, 'primary')} />
                    <OrgRow label="参与部门" color="default" orgs={pickOrgs(detail.organizations, 'secondary')} />
                    <OrgRow label="负责部门" color="blue" orgs={pickOrgs(detail.organizations, 'leader')} />
                  </Space>
                ),
              },
            ]}
          />
        )}
      </Drawer>
    </PageContainer>
  )
}

function OrgRow({ label, color, orgs }: { label: string; color: string; orgs: OrgOf[] }) {
  return (
    <div>
      <div style={{ color: tokens.textPlaceholder, fontSize: 12, marginBottom: 4 }}>{label}</div>
      {orgs.length ? (
        <Space size={4} wrap>
          {orgs.map((o) => (
            <Tag key={o.id} color={color}>
              {o.name || '-'}
            </Tag>
          ))}
        </Space>
      ) : (
        <span style={{ color: tokens.textPlaceholder }}>-</span>
      )}
    </div>
  )
}

// pickOrgs 按关系类型筛出组织列表。
function pickOrgs(orgs: UserOrganizationItem[], type: string): OrgOf[] {
  return (orgs || [])
    .filter((o) => o.relationType === type)
    .map((o) => ({ id: o.organizationID, name: o.organizationName }))
}

// toTreeSelect 组织树 -> TreeSelect 数据
function toTreeSelect(list: OrganizationItem[]): any[] {
  return list.map((n) => ({
    title: n.name,
    value: n.organizationID,
    children: n.children?.length ? toTreeSelect(n.children) : undefined,
  }))
}
