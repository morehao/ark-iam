import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Drawer,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Tree,
  message,
} from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { DataNode } from 'antd/es/tree'
import { actionColumn, CODE_COL_WIDTH, COUNT_COL_WIDTH, NAME_COL_WIDTH, PageContainer, SourceTag, STATUS_COL_WIDTH, tableScrollX, TAG_COL_WIDTH, textColumn, timeColumn, tokens } from '@ark-iam/ui'
import type { MenuItem, TenantAppItem, TenantRoleItem } from '@ark-iam/types'
import {
  createTenantRole,
  deleteTenantRole,
  getTenantRoleMenus,
  getTenantRolePageList,
  updateTenantRole,
  updateTenantRoleMenus,
} from '../../api/role'
import { getTenantApps } from '../../api/menu'

/** 系统管理类型展示：admin→管理员角色，normal→普通角色 */
function adminTypeText(adminType?: string) {
  switch (adminType) {
    case 'admin':
      return <Tag color="red">管理员角色</Tag>
    case 'normal':
    default:
      return <Tag>普通角色</Tag>
  }
}

/** 是否为内置管理员角色（source=builtin && admin_type=admin） */
function isBuiltinAdminRole(r?: TenantRoleItem): boolean {
  return !!r && r.source === 'builtin' && r.adminType === 'admin'
}

// 菜单树 -> Tree 数据；非内置管理员角色剔除 visibility=admin 节点（授权约束，前端兜底）
function toMenuTree(list: MenuItem[], builtinAdmin: boolean): DataNode[] {
  return list
    .filter((m) => builtinAdmin || m.visibility !== 'admin')
    .map((m) => ({
      key: m.menuID,
      title: m.name,
      children: m.children?.length ? toMenuTree(m.children, builtinAdmin) : undefined,
    }))
}

export default function TenantRolePage() {
  const [data, setData] = useState<TenantRoleItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [appFilter, setAppFilter] = useState<string | undefined>()
  const [apps, setApps] = useState<TenantAppItem[]>([])

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantRoleItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  // 菜单授权 Drawer
  const [menuOpen, setMenuOpen] = useState(false)
  const [menuRole, setMenuRole] = useState<TenantRoleItem | null>(null)
  const [menuTree, setMenuTree] = useState<DataNode[]>([])
  const [checkedKeys, setCheckedKeys] = useState<React.Key[]>([])
  const [menuLoading, setMenuLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  // 应用选项（角色从属于应用）
  useEffect(() => {
    getTenantApps().then((resp) => setApps(resp?.list || [])).catch(() => {})
  }, [])

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTenantRolePageList({ page, pageSize, appID: appFilter, keyword: keyword || undefined })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword, appFilter])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (record: TenantRoleItem) => {
    setEditing(record)
    form.setFieldsValue({ name: record.name, code: record.code, description: record.description })
    setModalOpen(true)
  }

  const submitRole = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateTenantRole({ roleID: editing.roleID, ...values })
        message.success('保存成功')
      } else {
        await createTenantRole(values)
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

  const handleDelete = async (record: TenantRoleItem) => {
    await deleteTenantRole(record.roleID)
    message.success('删除成功')
    void fetchData()
  }

  const openMenuAuth = async (record: TenantRoleItem) => {
    setMenuRole(record)
    setMenuOpen(true)
    setMenuLoading(true)
    setCheckedKeys([])
    try {
      const resp = await getTenantRoleMenus(record.roleID)
      setMenuTree(toMenuTree(resp?.list || [], isBuiltinAdminRole(record)))
      setCheckedKeys(resp?.menuIDs || [])
    } catch {
      /* 拦截器已提示 */
    } finally {
      setMenuLoading(false)
    }
  }

  const saveMenuAuth = async () => {
    if (!menuRole) return
    setSaving(true)
    try {
      await updateTenantRoleMenus(menuRole.roleID, checkedKeys.map(String))
      message.success('菜单权限已更新')
      setMenuOpen(false)
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSaving(false)
    }
  }

  const columns: ColumnsType<TenantRoleItem> = [
    textColumn<TenantRoleItem>({ title: '角色名称', dataIndex: 'name', width: NAME_COL_WIDTH }),
    { title: '所属应用', dataIndex: 'appName', key: 'appName', width: 120, render: (_: string, r) => <Tag>{r.appName || '系统角色'}</Tag> },
    textColumn<TenantRoleItem>({ title: '角色编码', dataIndex: 'code', width: CODE_COL_WIDTH, monospace: true }),
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: TAG_COL_WIDTH,
      render: (v: string) => <SourceTag value={v} />,
    },
    {
      title: '角色类型',
      dataIndex: 'adminType',
      key: 'adminType',
      width: STATUS_COL_WIDTH,
      render: (v: string) => adminTypeText(v),
    },
    textColumn<TenantRoleItem>({ title: '描述', dataIndex: 'description', width: 160 }),
    { title: '成员数', dataIndex: 'memberCount', key: 'memberCount', width: COUNT_COL_WIDTH, render: (v: number) => v || 0 },
    { title: '授权菜单', dataIndex: 'menuCount', key: 'menuCount', width: COUNT_COL_WIDTH, render: (v: number) => v || 0 },
    timeColumn<TenantRoleItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<TenantRoleItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<TenantRoleItem>({
      max: 3,
      actions: (r) => {
        const isBuiltin = r.source === 'builtin'
        return [
          { key: 'menu', label: '菜单权限', onClick: () => void openMenuAuth(r) },
          { key: 'edit', label: '编辑', disabled: isBuiltin, onClick: () => openEdit(r) },
          {
            key: 'delete',
            label: '删除',
            danger: true,
            disabled: isBuiltin,
            confirm: '确认删除该角色？（级联清理成员/菜单关联）',
            onClick: () => void handleDelete(r),
          },
        ]
      },
    }),
  ]

  return (
    <PageContainer
      title="角色管理"
      description="租户内的角色定义与授权：角色从属于应用，菜单授权为该应用的控制台菜单；成员由用户侧分配"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建角色
          </Button>
        </Space>
      }
    >
      <Space style={{ marginBottom: 16 }} wrap>
        <Select
          allowClear
          placeholder="所属应用"
          style={{ width: 180 }}
          value={appFilter}
          onChange={(v) => {
            setAppFilter(v)
            setPage(1)
          }}
          options={apps.map((a) => ({ label: a.name, value: a.appID }))}
        />
        <Input.Search
          allowClear
          placeholder="按角色名称/编码搜索"
          prefix={<SearchOutlined />}
          style={{ width: 260 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
      </Space>

      <Table<TenantRoleItem>
        rowKey="roleID"
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

      {/* 新建 / 编辑角色 */}
      <Modal
        title={editing ? '编辑角色' : '新建角色'}
        open={modalOpen}
        onOk={() => void submitRole()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          {/* 只读：来源 + 系统管理类型（新建固定 custom/normal，编辑按记录回显） */}
          <div style={{ display: 'flex', gap: 24, marginBottom: 20, color: tokens.textSecondary, fontSize: 13 }}>
            <span>
              来源：<SourceTag value={editing ? editing.source : 'custom'} />
            </span>
            <span>
              角色类型：{adminTypeText(editing?.adminType)}
            </span>
          </div>
          {!editing && (
            <Form.Item name="appID" label="所属应用" rules={[{ required: true, message: '请选择所属应用' }]}>
              <Select placeholder="选择该角色归属的应用" options={apps.map((a) => ({ label: a.name, value: a.appID }))} />
            </Form.Item>
          )}
          <Form.Item name="name" label="角色名称" rules={[{ required: true, message: '请输入角色名称' }]}>
            <Input placeholder="如：组织管理员" />
          </Form.Item>
          <Form.Item name="code" label="角色编码" rules={[{ required: true, message: '请输入角色编码' }]}>
            <Input placeholder="如：org_admin（应用内唯一）" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="选填" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 菜单权限授权 Drawer */}
      <Drawer
        title={`菜单权限 - ${menuRole?.name || ''}${menuRole?.appName ? `（${menuRole.appName}）` : ''}`}
        width={420}
        open={menuOpen}
        onClose={() => setMenuOpen(false)}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setMenuOpen(false)}>取消</Button>
            <Button type="primary" loading={saving} onClick={() => void saveMenuAuth()}>
              保存授权
            </Button>
          </Space>
        }
      >
        <div style={{ marginBottom: 12, color: tokens.textPlaceholder, fontSize: 12 }}>
          勾选该角色可访问的租户控制台页面（父级自动级联）
        </div>
        {menuLoading ? (
          <div style={{ padding: 40, textAlign: 'center', color: tokens.textPlaceholder }}>加载中...</div>
        ) : (
          <Tree
            checkable
            defaultExpandAll
            treeData={menuTree}
            checkedKeys={checkedKeys}
            onCheck={(keys) => setCheckedKeys(Array.isArray(keys) ? keys : keys.checked)}
            selectable={false}
          />
        )}
      </Drawer>
    </PageContainer>
  )
}
