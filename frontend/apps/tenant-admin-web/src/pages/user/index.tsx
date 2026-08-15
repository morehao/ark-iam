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
import { PageContainer } from '@ark-iam/ui'
import type { OrganizationItem, TenantRoleItem, TenantUserDetail, TenantUserItem, UserOrganizationItem } from '@ark-iam/types'
import {
  createTenantUser,
  getTenantUserDetail,
  getTenantUserPageList,
  resetTenantUserPassword,
  updateTenantUser,
  updateTenantUserRoles,
} from '../../api/user'
import { getOrganizationTree, updateUserOrganizations } from '../../api/organization'
import { getTenantRolePageList } from '../../api/role'
import { fmtTime } from '../../components/common'

export default function TenantUserPage() {
  const [data, setData] = useState<TenantUserItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [suspended, setSuspended] = useState<boolean | undefined>()

  // 创建 / 编辑
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantUserItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  // 详情 Drawer
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<TenantUserDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  // 组织归属（详情 Tab）
  const [orgTree, setOrgTree] = useState<OrganizationItem[]>([])
  const [orgIDs, setOrgIDs] = useState<string[]>([])

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
      const resp = await getTenantUserPageList({ page, pageSize, keyword: keyword || undefined, isSuspended: suspended })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword, suspended])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  // 组织树（创建表单部门下拉 + 详情组织归属共用）
  useEffect(() => {
    getOrganizationTree().then((resp) => setOrgTree(resp?.list || [])).catch(() => {})
  }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (record: TenantUserItem) => {
    setEditing(record)
    form.setFieldsValue({ name: record.name, avatar: record.avatar, isSuspended: record.isSuspended })
    setModalOpen(true)
  }

  const submitUser = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateTenantUser({ userID: editing.userID, name: values.name, avatar: values.avatar, isSuspended: values.isSuspended })
        message.success('保存成功')
      } else {
        await createTenantUser({ ...values, organizationIDs: [values.organizationID] })
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
      const [d, orgs, roles] = await Promise.all([
        getTenantUserDetail(record.userID),
        getOrganizationTree(),
        getTenantRolePageList({ page: 1, pageSize: 100 }),
      ])
      setDetail(d)
      setOrgTree(orgs?.list || [])
      setOrgIDs((d.organizations || []).filter((o) => o.relationType === 'member').map((o) => o.organizationID))
      setRoleOptions(roles?.list || [])
      setRoleIDs((d.roles || []).map((r) => r.roleID))
    } catch {
      /* 拦截器已提示 */
    } finally {
      setDetailLoading(false)
    }
  }

  const saveOrgAssignments = async () => {
    if (!detail) return
    // 业务约束：用户必须从属于至少一个部门，禁止清空全部组织归属
    if (!orgIDs.length) {
      message.error('用户必须从属于至少一个部门')
      return
    }
    await updateUserOrganizations(detail.userID, orgIDs)
    message.success('组织归属已更新')
    void openDetail(detail)
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
            <span style={{ fontSize: 12, color: '#94a3b8' }}>@{r.username || '-'}</span>
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
      render: (v: boolean) => (v ? <Tag color="red">挂起</Tag> : <Tag color="green">正常</Tag>),
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
          <Button type="link" size="small" onClick={() => openEdit(r)}>
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
      description="租户内的用户目录：创建账号、组织归属、角色分配与账号状态管理"
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

      {/* 新建 / 编辑用户 */}
      <Modal
        title={editing ? '编辑用户' : '新建用户'}
        open={modalOpen}
        onOk={() => void submitUser()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          {!editing && (
            <>
              <Form.Item name="name" label="姓名" rules={[{ required: true, message: '请输入姓名' }]}>
                <Input placeholder="如：张三（无匹配自然人时按此姓名创建）" />
              </Form.Item>
              <Form.Item name="organizationID" label="部门" rules={[{ required: true, message: '请选择部门' }]}>
                <TreeSelect
                  treeData={toTreeSelect(orgTree)}
                  treeDefaultExpandAll
                  placeholder="选择所属部门（同时建立组织归属）"
                />
              </Form.Item>
              <Form.Item name="primaryEmail" label="邮箱">
                <Input placeholder="可空；命中已有自然人则复用" />
              </Form.Item>
              <Form.Item name="primaryPhone" label="手机号">
                <Input placeholder="可空" />
              </Form.Item>
              <Form.Item name="password" label="初始密码">
                <Input.Password placeholder="可空；提供后该用户可登录" />
              </Form.Item>
              <Form.Item name="isSuspended" label="状态" valuePropName="checked" initialValue={false}>
                <Switch checkedChildren="挂起" unCheckedChildren="正常" />
              </Form.Item>
            </>
          )}
          {editing && (
            <>
              <Form.Item name="name" label="姓名" rules={[{ required: true, message: '请输入姓名' }]}>
                <Input />
              </Form.Item>
              <Form.Item name="avatar" label="头像URL">
                <Input placeholder="可空" />
              </Form.Item>
              <Form.Item name="isSuspended" label="状态" valuePropName="checked">
                <Switch checkedChildren="挂起" unCheckedChildren="正常" />
              </Form.Item>
            </>
          )}
        </Form>
      </Modal>

      {/* 详情 Drawer：基础信息 / 组织归属 / 角色 */}
      <Drawer
        title="用户详情"
        width={560}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        destroyOnClose={false}
      >
        {detailLoading ? (
          <div style={{ padding: 60, textAlign: 'center', color: '#94a3b8' }}>加载中...</div>
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
                        <span style={{ color: '#94a3b8' }}>@{detail.username || '-'}</span>
                      </Space>
                    </Space>
                    <div>
                      <Tag>{detail.isSuspended ? '挂起' : '正常'}</Tag>
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
                label: '组织归属',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <TreeSelect
                      treeData={toTreeSelect(orgTree)}
                      value={orgIDs}
                      onChange={setOrgIDs}
                      multiple
                      allowClear
                      treeDefaultExpandAll
                      style={{ width: '100%' }}
                      placeholder="勾选所属组织（首个为主组织）"
                    />
                    <div style={{ color: '#94a3b8', fontSize: 12 }}>
                      首个为主组织（主归属）；保存时全量替换成员关系
                    </div>
                    <Button type="primary" onClick={() => void saveOrgAssignments()}>
                      保存组织归属
                    </Button>
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
                    <div style={{ color: '#94a3b8', fontSize: 12 }}>当前已分配角色：{(detail.roles || []).map((r) => r.name).join('、') || '无'}</div>
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

// toTreeSelect 组织树 -> TreeSelect 数据
function toTreeSelect(list: OrganizationItem[]): any[] {
  return list.map((n) => ({
    title: n.name,
    value: n.organizationID,
    children: n.children?.length ? toTreeSelect(n.children) : undefined,
  }))
}

export type { UserOrganizationItem }
