import { useCallback, useEffect, useState } from 'react'
import {
  Avatar,
  Button,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
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
import type { OrganizationItem, TenantRoleItem, TenantUserDetail, TenantUserItem, UserOrganizationItem } from '@ark-iam/types'
import {
  createTenantUser,
  getTenantUserDetail,
  getTenantUserPageList,
  resetTenantUserPassword,
  updateTenantUser,
  updateTenantUserRoles,
} from '../../api/user'
import { getOrganizationTree } from '../../api/organization'
import { getTenantRolePageList } from '../../api/role'

// 组织关系类型 -> 展示标签
const RELATION_TAG: Record<string, { color: string; label: string }> = {
  primary: { color: 'gold', label: '主组织' },
  secondary: { color: 'blue', label: '参与组织' },
  leader: { color: 'green', label: '负责组织' },
}

export default function TenantUserPage() {
  const [data, setData] = useState<TenantUserItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [suspended, setSuspended] = useState<boolean | undefined>()
  const [organizationID, setOrganizationID] = useState<string>()

  // 创建 / 编辑
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantUserItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)
  // 编辑前的组织关系快照：仅在组织关系变化时才随 PATCH 提交，避免无谓地重置 joined_at
  const [orgBefore, setOrgBefore] = useState<{ primaryOrgID?: string; secondaryOrgIDs: string[]; leaderOrgIDs: string[] }>({
    secondaryOrgIDs: [],
    leaderOrgIDs: [],
  })

  // 详情 Drawer
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<TenantUserDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  // 组织树（创建/编辑表单组织下拉）
  const [orgTree, setOrgTree] = useState<OrganizationItem[]>([])

  // 角色（详情 Tab）
  const [roleOptions, setRoleOptions] = useState<TenantRoleItem[]>([])
  const [roleIDs, setRoleIDs] = useState<string[]>([])

  // 重置密码
  const [pwdOpen, setPwdOpen] = useState(false)
  const [pwdTarget, setPwdTarget] = useState<TenantUserItem | null>(null)
  const [pwdForm] = Form.useForm()

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTenantUserPageList({
        page,
        pageSize,
        keyword: keyword || undefined,
        isSuspended: suspended,
        organizationID: organizationID || undefined,
      })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword, suspended, organizationID])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  // 组织树（创建/编辑表单组织下拉共用）
  useEffect(() => {
    getOrganizationTree().then((resp) => setOrgTree(resp?.list || [])).catch(() => {})
  }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  // 编辑：先取详情回填（组织关系在编辑弹窗中一并维护：主/参与/负责组织）
  const openEdit = async (record: TenantUserItem) => {
    const detail = await getTenantUserDetail(record.userID).catch(() => null)
    if (!detail) return
    const nextPrimary = pickOrgs(detail.organizations, 'primary')[0]?.organizationID
    const nextSecondary = pickOrgs(detail.organizations, 'secondary').map((o) => o.organizationID)
    const nextLeader = pickOrgs(detail.organizations, 'leader').map((o) => o.organizationID)
    setOrgBefore({ primaryOrgID: nextPrimary, secondaryOrgIDs: nextSecondary, leaderOrgIDs: nextLeader })
    setEditing(record)
    form.setFieldsValue({
      name: detail.name,
      username: detail.username,
      primaryEmail: detail.primaryEmail,
      primaryPhone: detail.primaryPhone,
      avatar: detail.avatar,
      isSuspended: detail.isSuspended,
      primaryOrgID: nextPrimary,
      secondaryOrgIDs: nextSecondary,
      leaderOrgIDs: nextLeader,
    })
    setModalOpen(true)
  }

  const submitUser = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        const patch: {
          userID: string
          name: string
          username: string
          primaryEmail: string
          primaryPhone: string
          avatar: string
          isSuspended: boolean
          primaryOrgID?: string
          secondaryOrgIDs?: string[]
          leaderOrgIDs?: string[]
        } = {
          userID: editing.userID,
          name: values.name,
          username: values.username || '',
          primaryEmail: values.primaryEmail || '',
          primaryPhone: values.primaryPhone || '',
          avatar: values.avatar,
          isSuspended: values.isSuspended,
        }
        // 组织关系仅在变化时提交（PATCH 局部更新语义：不传=不变），避免无谓重写归属行
        const secondary = values.secondaryOrgIDs || []
        const leader = values.leaderOrgIDs || []
        if (values.primaryOrgID !== orgBefore.primaryOrgID) patch.primaryOrgID = values.primaryOrgID
        if (!sameSet(secondary, orgBefore.secondaryOrgIDs)) patch.secondaryOrgIDs = secondary
        if (!sameSet(leader, orgBefore.leaderOrgIDs)) patch.leaderOrgIDs = leader
        await updateTenantUser(patch)
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
      setModalOpen(false)
      void fetchData()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const toggleSuspended = async (record: TenantUserItem, checked: boolean) => {
    await updateTenantUser({ userID: record.userID, isSuspended: checked })
    message.success(checked ? '已挂起' : '已恢复')
    void fetchData()
  }

  const openDetail = async (record: TenantUserItem) => {
    setDetailOpen(true)
    setDetailLoading(true)
    setDetail(null)
    try {
      const [d, roles] = await Promise.all([
        getTenantUserDetail(record.userID),
        getTenantRolePageList({ page: 1, pageSize: 100 }),
      ])
      setDetail(d)
      setRoleOptions(roles?.list || [])
      setRoleIDs((d.roles || []).map((r) => r.roleID))
    } catch {
      /* 拦截器已提示 */
    } finally {
      setDetailLoading(false)
    }
  }

  const saveRoleAssignments = async () => {
    if (!detail) return
    await updateTenantUserRoles(detail.userID, roleIDs)
    message.success('角色已更新')
    void openDetail(detail)
  }

  const openResetPassword = (record: TenantUserItem) => {
    setPwdTarget(record)
    pwdForm.resetFields()
    setPwdOpen(true)
  }

  const submitResetPassword = async () => {
    try {
      const values = await pwdForm.validateFields()
      if (!pwdTarget) return
      await resetTenantUserPassword(pwdTarget.userID, values.password)
      message.success('密码已重置')
      setPwdOpen(false)
    } catch {
      /* 校验或请求失败 */
    }
  }

  const columns: ColumnsType<TenantUserItem> = [
    {
      title: '用户',
      key: 'user',
      width: 210,
      render: (_, r) => (
        <Space>
          <Avatar size={30}>{r.name?.charAt(0)?.toUpperCase() || 'U'}</Avatar>
          <Space direction="vertical" size={0}>
            <span style={{ fontWeight: 500 }}>{r.name || '-'}</span>
            <span style={{ fontSize: 12, color: tokens.textPlaceholder }}>@{r.username || '-'}</span>
          </Space>
        </Space>
      ),
    },
    { title: '邮箱', dataIndex: 'primaryEmail', key: 'primaryEmail', render: (v: string) => v || '-' },
    { title: '手机号', dataIndex: 'primaryPhone', key: 'primaryPhone', width: 130, render: (v: string) => v || '-' },
    { title: '主组织', dataIndex: 'primaryOrgName', key: 'primaryOrgName', width: 150, render: (v: string) => v || '-' },
    { title: '角色数', dataIndex: 'roleCount', key: 'roleCount', width: 80, render: (v: number) => v || 0 },
    {
      title: '状态',
      dataIndex: 'isSuspended',
      key: 'isSuspended',
      width: 90,
      render: (v: boolean) => <SuspendedTag value={v} />,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (v?: number) => fmtTime(v),
    },
    {
      title: '操作',
      key: 'action',
      width: 260,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => void openDetail(r)}>
            详情
          </Button>
          <Button type="link" size="small" onClick={() => void openEdit(r)}>
            编辑
          </Button>
          <Popconfirm title={r.isSuspended ? '确认恢复该用户？' : '确认挂起该用户？'} onConfirm={() => void toggleSuspended(r, !r.isSuspended)}>
            <Button type="link" size="small" danger={!r.isSuspended}>
              {r.isSuspended ? '恢复' : '挂起'}
            </Button>
          </Popconfirm>
          <Button type="link" size="small" onClick={() => openResetPassword(r)}>
            重置密码
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title="用户管理"
      description="租户内的用户目录：创建账号、组织归属（主/参与/负责组织）、角色分配与账号状态管理"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建用户
          </Button>
        </Space>
      }
    >
      <Space style={{ marginBottom: 16 }} wrap>
        <TreeSelect
          allowClear
          treeData={toTreeSelect(orgTree)}
          treeDefaultExpandAll
          placeholder="按组织筛选（恰在该组织）"
          style={{ width: 220 }}
          value={organizationID}
          onChange={(v) => {
            setOrganizationID(v)
            setPage(1)
          }}
        />
        <Select
          allowClear
          placeholder="状态"
          style={{ width: 110 }}
          value={suspended}
          onChange={(v) => {
            setSuspended(v)
            setPage(1)
          }}
          options={[
            { label: '正常', value: false },
            { label: '挂起', value: true },
          ]}
        />
        <Input.Search
          allowClear
          placeholder="姓名/用户名/邮箱/手机"
          prefix={<SearchOutlined />}
          style={{ width: 260 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
      </Space>

      <Table<TenantUserItem>
        rowKey="userID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1100 }}
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

      {/* 新建 / 编辑用户：基础信息 + 组织关系（主/参与/负责组织） + 账号状态 */}
      <Modal
        title={editing ? '编辑用户' : '新建用户'}
        open={modalOpen}
        onOk={() => void submitUser()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={620}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="姓名" rules={[{ required: true, message: '请输入姓名' }]}>
            <Input placeholder="如：张三（无匹配自然人时按此姓名创建）" />
          </Form.Item>
          <Form.Item name="primaryOrgID" label="主组织" rules={[{ required: true, message: '请选择主组织' }]}>
            <TreeSelect
              treeData={toTreeSelect(orgTree)}
              treeDefaultExpandAll
              placeholder="选择主组织（行政归属，唯一；同时建立组织归属）"
            />
          </Form.Item>
          <Form.Item name="secondaryOrgIDs" label="参与组织">
            <TreeSelect
              treeData={toTreeSelect(orgTree)}
              treeDefaultExpandAll
              multiple
              allowClear
              placeholder="选择参与组织（可多个，跨组织协作）"
            />
          </Form.Item>
          <Form.Item name="leaderOrgIDs" label="负责组织">
            <TreeSelect
              treeData={toTreeSelect(orgTree)}
              treeDefaultExpandAll
              multiple
              allowClear
              placeholder="选择负责组织（可多个，但每组织至多一位负责人）"
            />
          </Form.Item>
          <Form.Item
            name="primaryEmail"
            label="邮箱"
            dependencies={['primaryPhone']}
            rules={[
              {
                validator: (_, v) => {
                  const phone = form.getFieldValue('primaryPhone')
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
              <Input.Password placeholder="可空；提供后该用户可登录" />
            </Form.Item>
          )}
          {editing && (
            <Form.Item name="avatar" label="头像URL">
              <Input placeholder="可空" />
            </Form.Item>
          )}
          <Form.Item name="isSuspended" label="状态" valuePropName="checked" initialValue={false}>
            <Switch checkedChildren="挂起" unCheckedChildren="正常" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情 Drawer：基础信息 / 组织关系 / 角色 */}
      <Drawer title="用户详情" width={560} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose={false}>
        {detailLoading ? (
          <div style={{ padding: 60, textAlign: 'center', color: tokens.textPlaceholder }}>加载中...</div>
        ) : detail ? (
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
                      <SuspendedTag value={detail.isSuspended} />
                      <span style={{ marginLeft: 8 }}>用户ID：{detail.userID}</span>
                    </div>
                    <div>邮箱：{detail.primaryEmail || '-'}</div>
                    <div>手机号：{detail.primaryPhone || '-'}</div>
                    <div>创建时间：{fmtTime(detail.createdAt)}</div>
                  </Space>
                ),
              },
              {
                key: 'org',
                label: '组织关系',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <OrgRow label="主组织" relation="primary" orgs={pickOrgs(detail.organizations, 'primary')} />
                    <OrgRow label="参与组织" relation="secondary" orgs={pickOrgs(detail.organizations, 'secondary')} />
                    <OrgRow label="负责组织" relation="leader" orgs={pickOrgs(detail.organizations, 'leader')} />
                    <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>
                      组织关系的调整请使用「编辑」：主/参与/负责组织在编辑弹窗中全量维护
                    </div>
                  </Space>
                ),
              },
              {
                key: 'role',
                label: '角色',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Select
                      mode="multiple"
                      allowClear
                      placeholder="选择角色（全量替换）"
                      style={{ width: '100%' }}
                      value={roleIDs}
                      onChange={setRoleIDs}
                      options={roleOptions.map((r) => ({ label: `${r.name}（${r.code}）`, value: r.roleID }))}
                    />
                    <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>当前已分配角色：{(detail.roles || []).map((r) => r.name).join('、') || '无'}</div>
                    <Button type="primary" onClick={() => void saveRoleAssignments()}>
                      保存角色
                    </Button>
                  </Space>
                ),
              },
            ]}
          />
        ) : null}
      </Drawer>

      {/* 重置密码 */}
      <Modal title={`重置密码 - ${pwdTarget?.name || ''}`} open={pwdOpen} onOk={() => void submitResetPassword()} onCancel={() => setPwdOpen(false)} destroyOnClose>
        <Form form={pwdForm} layout="vertical">
          <Form.Item name="password" label="新密码" rules={[{ required: true, message: '请输入新密码' }]}>
            <Input.Password placeholder="重置后用户使用新密码登录" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}

// pickOrgs 按关系类型筛出组织列表。
function pickOrgs(orgs: UserOrganizationItem[], type: string): UserOrganizationItem[] {
  return (orgs || []).filter((o) => o.relationType === type)
}

// OrgRow 详情中的组织关系行（主/参与/负责）。
function OrgRow({ label, relation, orgs }: { label: string; relation: string; orgs: UserOrganizationItem[] }) {
  const tag = RELATION_TAG[relation]
  return (
    <div>
      <div style={{ color: tokens.textPlaceholder, fontSize: 12, marginBottom: 4 }}>{label}</div>
      {orgs.length ? (
        <Space size={4} wrap>
          {orgs.map((o) => (
            <Tag key={o.organizationID} color={tag?.color}>
              {o.organizationName || '-'}
            </Tag>
          ))}
        </Space>
      ) : (
        <span style={{ color: tokens.textPlaceholder }}>-</span>
      )}
    </div>
  )
}

// toTreeSelect 组织树 -> TreeSelect 数据
function toTreeSelect(list: OrganizationItem[]): any[] {
  return list.map((n) => ({
    title: n.name,
    value: n.organizationID,
    children: n.children?.length ? toTreeSelect(n.children) : undefined,
  }))
}

// sameSet 两个数组按集合语义比较（忽略顺序与重复）。
function sameSet(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false
  const set = new Set(b)
  return a.every((v) => set.has(v))
}
