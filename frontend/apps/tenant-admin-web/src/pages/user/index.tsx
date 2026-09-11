import { useCallback, useEffect, useState } from 'react'
import {
  Avatar,
  Button,
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
import { fmtTime, InitialPasswordModal, NameLink, PageContainer, RowActions, SuspendedTag, timeColumn, tokens } from '@ark-iam/ui'
import type {
  OrganizationItem,
  TenantApiKeyItem,
  TenantMachineUserDetail,
  TenantMachineUserItem,
  TenantMachineUserUpdateReq,
  TenantUserDetail,
  TenantUserItem,
} from '@ark-iam/types'
import {
  createTenantUser,
  getTenantUserDetail,
  getTenantUserPageList,
  resetTenantUserPassword,
  updateTenantUser,
} from '../../api/user'
import {
  createMachineUser,
  deleteMachineUser,
  getMachineUserDetail,
  getMachineUserPageList,
  updateMachineUser,
  updateMachineUserStatus,
} from '../../api/machineUser'
import { getApiKeyPageList } from '../../api/apiKey'
import { getOrganizationTree } from '../../api/organization'
import RoleAssignEditor from '../../components/RoleAssignEditor'
import UserIdentityTab from '../../components/UserIdentityTab'
import UserLoginLogTab from '../../components/UserLoginLogTab'
import { KeyStateTag } from '../apiKey/KeyState'

// 组织关系类型 -> 展示标签
const RELATION_TAG: Record<string, { color: string; label: string }> = {
  primary: { color: 'gold', label: '主组织' },
  secondary: { color: 'blue', label: '参与组织' },
  leader: { color: 'green', label: '负责组织' },
}

// ==================== Tab：用户（真实用户，保持原逻辑不变） ====================
function UsersPane() {
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

  // 授权角色（列表行操作 Modal：按应用授权，逻辑收敛于共享组件 RoleAssignEditor）
  const [roleTarget, setRoleTarget] = useState<TenantUserItem | null>(null)

  // 一次性初始/临时密码（建成员或重置密码后展示，关闭即不可再查）
  const [initialPassword, setInitialPassword] = useState('')

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
        const created = await createTenantUser({
          name: values.name,
          username: values.username,
          primaryEmail: values.primaryEmail,
          primaryPhone: values.primaryPhone,
          isSuspended: values.isSuspended,
          organizationIDs: [values.primaryOrgID],
          secondaryOrgIDs: values.secondaryOrgIDs || [],
          leaderOrgIDs: values.leaderOrgIDs || [],
        })
        setModalOpen(false)
        void fetchData()
        if (created.initialPassword) {
          // 初始临时密码仅在创建响应返回一次；该成员首次登录必须改密
          message.success('创建成功')
          setInitialPassword(created.initialPassword)
        } else {
          Modal.info({
            title: '成员已创建',
            content: '该邮箱/手机已存在统一身份账号，密码未被改动；如需登录凭据，可对该成员执行「重置密码」。',
          })
        }
        return
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
      const d = await getTenantUserDetail(record.userID)
      setDetail(d)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setDetailLoading(false)
    }
  }

  /** 重置成员密码：二次确认由 RowActions 的行内确认承担；成功后一次性展示新临时密码 */
  const resetPassword = async (record: TenantUserItem) => {
    try {
      const resp = await resetTenantUserPassword(record.userID)
      setInitialPassword(resp.initialPassword)
    } catch {
      /* 拦截器已提示 */
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
            <NameLink value={r.name || '-'} onClick={() => void openDetail(r)} />
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
    timeColumn<TenantUserItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<TenantUserItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, r) => (
        <RowActions
          actions={[
            { key: 'edit', label: '编辑', onClick: () => void openEdit(r) },
            { key: 'roles', label: '授权角色', onClick: () => setRoleTarget(r) },
            { key: 'resetPwd', label: '重置密码', confirm: '重置后系统生成新的临时密码并仅展示一次，该成员既有会话将失效。确认重置？', onClick: () => void resetPassword(r) },
            {
              key: 'toggle',
              label: r.isSuspended ? '恢复' : '挂起',
              danger: !r.isSuspended,
              confirm: r.isSuspended ? '确认恢复该用户？' : '确认挂起该用户？',
              onClick: () => void toggleSuspended(r, !r.isSuspended),
            },
          ]}
        />
      ),
    },
  ]

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
        <Space wrap>
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
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建用户
          </Button>
        </Space>
      </div>

      <Table<TenantUserItem>
        rowKey="userID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1400 }}
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
            <Form.Item label="初始密码" extra="由系统生成临时密码，创建后仅在弹窗中展示一次；该成员首次登录必须修改密码">
              <Input value="创建后自动生成" disabled />
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
                children: <RoleAssignEditor kind="user" subjectID={detail.userID} onSaved={() => void fetchData()} />,
              },
              {
                key: 'identity',
                label: '第三方身份',
                children: <UserIdentityTab userID={detail.userID} />,
              },
              {
                key: 'loginLog',
                label: '登录日志',
                children: <UserLoginLogTab userID={detail.userID} />,
              },
            ]}
          />
        ) : null}
      </Drawer>

      {/* 一次性初始/临时密码（建成员、重置密码共用同一交互） */}
      <InitialPasswordModal
        password={initialPassword}
        description="该成员首次登录必须修改密码；其既有会话已失效。"
        onClose={() => setInitialPassword('')}
      />



      {/* 授权角色（列表行操作）：按应用授权，内容收敛于共享组件 */}
      <Modal
        title={`授权角色 - ${roleTarget?.name || ''}`}
        open={!!roleTarget}
        onCancel={() => setRoleTarget(null)}
        footer={null}
        destroyOnClose
        width={560}
      >
        {roleTarget && (
          <RoleAssignEditor
            kind="user"
            subjectID={roleTarget.userID}
            onSaved={() => {
              setRoleTarget(null)
              void fetchData()
            }}
          />
        )}
      </Modal>
    </>
  )
}

// ==================== Tab：服务账号（机器主体，user_type=machine） ====================
function ServiceAccountsPane() {
  const [data, setData] = useState<TenantMachineUserItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [isSuspended, setIsSuspended] = useState<boolean | undefined>()

  // 创建 / 编辑
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantMachineUserItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)
  // 编辑前的组织快照：仅在组织关系变化时才随 PUT 提交（可空字段语义：不传=不变）
  const [orgBefore, setOrgBefore] = useState<{ primaryOrgID?: string; secondaryOrgIDs: string[] }>({ secondaryOrgIDs: [] })

  // 详情 Drawer
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<TenantMachineUserDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  // 打开详情时的行记录：Drawer 内「编辑」复用表格行编辑逻辑
  const [activeRecord, setActiveRecord] = useState<TenantMachineUserItem | null>(null)

  // 组织树（创建/编辑表单组织下拉，与真实用户表单的组织选择一致）
  const [orgTree, setOrgTree] = useState<OrganizationItem[]>([])

  // 服务账号 API 密钥（详情内只读）
  const [machineKeys, setMachineKeys] = useState<TenantApiKeyItem[]>([])
  const [machineKeysLoading, setMachineKeysLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getMachineUserPageList({
        page,
        pageSize,
        name: keyword || undefined,
        isSuspended,
      })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword, isSuspended])

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

  // 编辑：先取详情回填（主/参与部门在编辑弹窗中一并维护）
  const openEdit = async (record: TenantMachineUserItem) => {
    const detail = await getMachineUserDetail(record.machineUserID).catch(() => null)
    if (!detail) return
    const nextPrimary = pickOrgs(detail.organizations, 'primary')[0]?.organizationID || detail.primaryOrgID || ''
    const nextSecondary = pickOrgs(detail.organizations, 'secondary').map((o) => o.organizationID)
    setOrgBefore({ primaryOrgID: nextPrimary || undefined, secondaryOrgIDs: nextSecondary })
    setEditing(record)
    form.setFieldsValue({
      name: detail.name,
      description: detail.description,
      primaryOrgID: nextPrimary || undefined,
      secondaryOrgIDs: nextSecondary,
    })
    setModalOpen(true)
  }

  const submitMachine = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        const secondary = values.secondaryOrgIDs || []
        const req: TenantMachineUserUpdateReq = {
          machineUserID: editing.machineUserID,
          name: values.name,
          description: values.description || '',
        }
        // 可空字段仅在变化时提交：primaryOrgID 传值=替换主部门；secondaryOrgIDs 传=全量替换（[]=清空）；不传=不变
        if (values.primaryOrgID !== orgBefore.primaryOrgID) req.primaryOrgID = values.primaryOrgID
        if (!sameSet(secondary, orgBefore.secondaryOrgIDs)) req.secondaryOrgIDs = secondary
        await updateMachineUser(req)
        message.success('保存成功')
      } else {
        await createMachineUser({
          name: values.name,
          description: values.description || '',
          organizationIDs: [values.primaryOrgID],
          secondaryOrgIDs: values.secondaryOrgIDs || [],
        })
        message.success('创建成功')
      }
      setModalOpen(false)
      void fetchData()
      // 在详情 Drawer 内编辑并保存后，同步刷新详情，保证部门/名称/描述展示最新
      if (detailOpen && editing && detail && detail.machineUserID === editing.machineUserID) {
        getMachineUserDetail(editing.machineUserID).then(setDetail).catch(() => {})
      }
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const toggleSuspended = async (record: TenantMachineUserItem, checked: boolean) => {
    await updateMachineUserStatus(record.machineUserID, checked)
    message.success(checked ? '已挂起' : '已启用')
    void fetchData()
  }

  const handleDelete = async (record: TenantMachineUserItem) => {
    await deleteMachineUser(record.machineUserID)
    message.success('删除成功')
    void fetchData()
  }

  const openDetail = async (record: TenantMachineUserItem) => {
    setDetailOpen(true)
    setActiveRecord(record)
    setDetailLoading(true)
    setDetail(null)
    setMachineKeys([])
    setMachineKeysLoading(true)
    try {
      const [d, keys] = await Promise.all([
        getMachineUserDetail(record.machineUserID),
        getApiKeyPageList({ machineUserID: record.machineUserID, page: 1, pageSize: 50 }).catch(() => ({ list: [], total: 0 })),
      ])
      setDetail(d)
      setMachineKeys(keys?.list || [])
    } catch {
      /* 拦截器已提示 */
    } finally {
      setDetailLoading(false)
      setMachineKeysLoading(false)
    }
  }

  const columns: ColumnsType<TenantMachineUserItem> = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 180, render: (v: string, r) => <NameLink value={v} onClick={() => void openDetail(r)} /> },
    { title: '主部门', dataIndex: 'primaryOrgName', key: 'primaryOrgName', width: 170, render: (v: string) => v || '-' },
    { title: '描述', dataIndex: 'description', key: 'description', render: (v: string) => v || '-' },
    {
      title: '状态',
      dataIndex: 'isSuspended',
      key: 'isSuspended',
      width: 90,
      render: (v: boolean) => <SuspendedTag value={v} />,
    },
    timeColumn<TenantMachineUserItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<TenantMachineUserItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, r) => (
        <RowActions
          actions={[
            { key: 'edit', label: '编辑', onClick: () => void openEdit(r) },
            {
              key: 'toggle',
              label: r.isSuspended ? '启用' : '挂起',
              danger: !r.isSuspended,
              confirm: r.isSuspended ? '确认恢复该服务账号？' : '确认挂起该服务账号？',
              onClick: () => void toggleSuspended(r, !r.isSuspended),
            },
            {
              key: 'delete',
              label: '删除',
              danger: true,
              confirm: '确认删除该服务账号？删除前须先删除其全部 API 密钥。',
              onClick: () => void handleDelete(r),
            },
          ]}
        />
      ),
    },
  ]

  const keyColumns: ColumnsType<TenantApiKeyItem> = [
    { title: '名称', dataIndex: 'name', key: 'name', render: (v: string) => v || '-' },
    {
      title: '前缀',
      dataIndex: 'keyPrefix',
      key: 'keyPrefix',
      width: 180,
      render: (v: string) => <span style={{ fontFamily: 'Consolas, Monaco, monospace', fontSize: 12 }}>{v || '-'}</span>,
    },
    { title: '状态', key: 'status', width: 90, render: (_: unknown, r) => <KeyStateTag {...r} /> },
    timeColumn<TenantApiKeyItem>({ title: '过期时间', dataIndex: 'expiredAt', placeholder: '永不过期' }),
    timeColumn<TenantApiKeyItem>({ title: '最近使用', dataIndex: 'lastUsedAt', relative: true, placeholder: '从未使用' }),
    timeColumn<TenantApiKeyItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<TenantApiKeyItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
  ]

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
        <Space wrap>
          <Input.Search
            allowClear
            placeholder="按名称搜索"
            prefix={<SearchOutlined />}
            style={{ width: 260 }}
            onSearch={(v) => {
              setKeyword(v)
              setPage(1)
            }}
          />
          <Select
            allowClear
            placeholder="状态"
            style={{ width: 110 }}
            value={isSuspended}
            onChange={(v) => {
              setIsSuspended(v)
              setPage(1)
            }}
            options={[
              { label: '启用', value: false },
              { label: '挂起', value: true },
            ]}
          />
        </Space>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建服务账号
          </Button>
        </Space>
      </div>

      <Table<TenantMachineUserItem>
        rowKey="machineUserID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1200 }}
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

      {/* 新建 / 编辑服务账号 */}
      <Modal
        title={editing ? '编辑服务账号' : '新建服务账号'}
        open={modalOpen}
        onOk={() => void submitMachine()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="如：CI 构建服务" />
          </Form.Item>
          <Form.Item name="primaryOrgID" label="主部门" rules={[{ required: true, message: '请选择主部门' }]}>
            <TreeSelect
              treeData={toTreeSelect(orgTree)}
              treeDefaultExpandAll
              placeholder="选择主部门（行政归属，唯一；服务账号必须从属部门）"
            />
          </Form.Item>
          <Form.Item name="secondaryOrgIDs" label="参与部门">
            <TreeSelect
              treeData={toTreeSelect(orgTree)}
              treeDefaultExpandAll
              multiple
              allowClear
              placeholder="选择参与部门（可多个，可留空）"
            />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="选填" />
          </Form.Item>
          <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>服务账号不可登录；必须从属主部门（可多条参与部门）；仅可被授予普通角色。</div>
        </Form>
      </Modal>

      {/* 详情 Drawer：基础信息 + 所属部门 + 已授权角色 + API 密钥只读列表 */}
      <Drawer title="服务账号详情" width={680} open={detailOpen} onClose={() => { setDetailOpen(false); setActiveRecord(null) }} destroyOnClose={false}>
        {detailLoading ? (
          <div style={{ padding: 60, textAlign: 'center', color: tokens.textPlaceholder }}>加载中...</div>
        ) : detail ? (
          <Tabs
            items={[
              { key: 'info', label: '基础信息', children: <MachineDetailInfo detail={detail} /> },
              {
                key: 'org',
                label: '所属部门',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <OrgRow label="主部门" relation="primary" orgs={pickOrgs(detail.organizations, 'primary')} />
                    <OrgRow label="参与部门" relation="secondary" orgs={pickOrgs(detail.organizations, 'secondary')} />
                    <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>
                      主部门唯一且服务账号必须从属部门；参与部门可多条（可清空）。部门调整请使用「编辑」（主/参与部门全量维护）。
                    </div>
                    <Button type="primary" onClick={() => activeRecord && void openEdit(activeRecord)}>
                      编辑
                    </Button>
                  </Space>
                ),
              },
              {
                key: 'role',
                label: '已授权角色',
                children: <RoleAssignEditor kind="machine" subjectID={detail.machineUserID} onSaved={() => void fetchData()} />,
              },
              {
                key: 'keys',
                label: 'API 密钥',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>以下为该服务账号的 API 密钥（只读）。密钥管理请前往「API密钥」模块。</div>
                    <Table<TenantApiKeyItem>
                      rowKey="keyID"
                      columns={keyColumns}
                      dataSource={machineKeys}
                      loading={machineKeysLoading}
                      size="small"
                      pagination={false}
                      scroll={{ x: 1000 }}
                    />
                  </Space>
                ),
              },
            ]}
          />
        ) : null}
      </Drawer>
    </>
  )
}

function MachineDetailInfo({ detail }: { detail: TenantMachineUserDetail }) {
  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Space>
        <Avatar size={48}>{detail.name?.charAt(0)?.toUpperCase() || 'S'}</Avatar>
        <Space direction="vertical" size={0}>
          <span style={{ fontWeight: 600, fontSize: 16 }}>{detail.name || '-'}</span>
          <span style={{ color: tokens.textPlaceholder }}>服务账号</span>
        </Space>
      </Space>
      <div>
        <SuspendedTag value={detail.isSuspended} />
        <span style={{ marginLeft: 8 }}>ID：{detail.machineUserID}</span>
      </div>
      <div>描述：{detail.description || '-'}</div>
      <div>创建时间：{fmtTime(detail.createdAt)}</div>
      <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>服务账号不可登录；所属部门见「所属部门」页签；仅可被授予普通角色。</div>
    </Space>
  )
}

// ==================== 页面容器：用户 / 服务账号 双 Tab ====================
export default function TenantUserPage() {
  return (
    <PageContainer title="用户管理" description="租户内的主体管理：真实用户（组织归属、角色分配与账号状态）与服务账号（机器主体，须从属主部门）">
      <Tabs
        defaultActiveKey="user"
        items={[
          { key: 'user', label: '用户', children: <UsersPane /> },
          { key: 'machine', label: '服务账号', children: <ServiceAccountsPane /> },
        ]}
      />
    </PageContainer>
  )
}

// pickOrgs 按关系类型筛出组织列表（真实用户与机器服务账号通用）。
function pickOrgs<T extends { relationType: string }>(orgs: T[], type: string): T[] {
  return (orgs || []).filter((o) => o.relationType === type)
}

// OrgRow 详情中的组织关系行（主/参与/负责）。
function OrgRow({ label, relation, orgs }: { label: string; relation: string; orgs: { organizationID: string; organizationName?: string }[] }) {
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
