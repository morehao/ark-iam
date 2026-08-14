import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Button,
  Divider,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
  Tree,
  TreeSelect,
  message,
} from 'antd'
import type { DataNode } from 'antd/es/tree'
import {
  AppstoreOutlined,
  FolderOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  ThunderboltOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons'
import { PageContainer, brand } from '@ark-iam/ui'
import { createMenu, deleteMenu, getApplicationPageList, getMenuTree, updateMenu } from '@ark-iam/api'
import type { ApplicationItem, MenuItem } from '@ark-iam/types'

interface MenuTreeNode {
  key: number
  title: string
  menu: MenuItem
  children?: MenuTreeNode[]
}

interface MenuFormValues {
  parentID: number
  name: string
  code: string
  path?: string
  icon?: string
  sort?: number
  type?: string
  component?: string
  redirect?: string
  permission?: string
  status?: string
  hidden: number
  externalLink: number
  keepAlive: number
}

type ModalMode = 'createRoot' | 'createChild' | 'edit'

const STORAGE_KEY = 'ark-iam:menu:appId'

const TYPE_META: Record<string, { label: string; icon: React.ReactNode; color: string }> = {
  directory: { label: '目录', icon: <FolderOutlined />, color: '#f59e0b' },
  menu: { label: '菜单', icon: <AppstoreOutlined />, color: '#4f6ef7' },
  button: { label: '按钮', icon: <ThunderboltOutlined />, color: '#7a5af8' },
}

function buildTreeData(list: MenuItem[]): MenuTreeNode[] {
  return list.map((item) => ({
    key: item.menuID,
    title: item.name,
    menu: item,
    children: item.children?.length ? buildTreeData(item.children) : undefined,
  }))
}

function collectKeys(list: MenuItem[]): number[] {
  const keys: number[] = []
  const walk = (items: MenuItem[]) => {
    items.forEach((item) => {
      keys.push(item.menuID)
      if (item.children?.length) walk(item.children)
    })
  }
  walk(list)
  return keys
}

function filterTree(nodes: MenuTreeNode[], keyword: string): MenuTreeNode[] {
  const kw = keyword.trim().toLowerCase()
  if (!kw) return nodes
  const hit = (m: MenuItem) =>
    m.name.toLowerCase().includes(kw) || m.code.toLowerCase().includes(kw) || (m.path || '').toLowerCase().includes(kw)
  const result: MenuTreeNode[] = []
  nodes.forEach((n) => {
    const children = n.children ? filterTree(n.children, keyword) : undefined
    const self = hit(n.menu)
    if (self || (children && children.length > 0)) {
      result.push({ ...n, children: self ? n.children : children })
    }
  })
  return result
}

export default function MenuList() {
  const [apps, setApps] = useState<ApplicationItem[]>([])
  const [appLoading, setAppLoading] = useState(false)
  const [selectedAppId, setSelectedAppId] = useState<number | undefined>()

  const [treeData, setTreeData] = useState<MenuTreeNode[]>([])
  const [expandedKeys, setExpandedKeys] = useState<number[]>([])
  const [treeLoading, setTreeLoading] = useState(false)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [mode, setMode] = useState<ModalMode>('createRoot')
  const [editing, setEditing] = useState<MenuItem | null>(null)
  const [parentMenu, setParentMenu] = useState<MenuItem | null>(null)
  const [form] = Form.useForm<MenuFormValues>()
  const [submitLoading, setSubmitLoading] = useState(false)

  const selectedApp = useMemo(
    () => apps.find((a) => a.appId === selectedAppId) || null,
    [apps, selectedAppId],
  )

  // 加载应用列表，用于应用切换器
  const loadApps = useCallback(async () => {
    setAppLoading(true)
    try {
      const resp = await getApplicationPageList({ page: 1, pageSize: 100 })
      const list = resp?.list || []
      setApps(list)
      const saved = Number(localStorage.getItem(STORAGE_KEY) || 0)
      const next = list.find((a) => a.appId === saved) || list[0]
      setSelectedAppId(next?.appId)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setAppLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadApps()
  }, [loadApps])

  // 按选中应用加载菜单树
  const fetchData = useCallback(async () => {
    if (!selectedAppId) return
    setTreeLoading(true)
    try {
      const resp = await getMenuTree(selectedAppId)
      const list = resp?.list || []
      setTreeData(buildTreeData(list))
      setExpandedKeys(collectKeys(list))
    } catch {
      /* 拦截器已提示 */
    } finally {
      setTreeLoading(false)
    }
  }, [selectedAppId])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleAppChange = (appId: number) => {
    setSelectedAppId(appId)
    localStorage.setItem(STORAGE_KEY, String(appId))
  }

  // 统计当前应用的菜单构成
  const stats = useMemo(() => {
    const all: MenuItem[] = []
    const walk = (items: MenuTreeNode[]) => {
      items.forEach((n) => {
        all.push(n.menu)
        if (n.children?.length) walk(n.children as MenuTreeNode[])
      })
    }
    walk(treeData)
    return {
      total: all.length,
      directory: all.filter((m) => m.type === 'directory').length,
      menu: all.filter((m) => m.type === 'menu').length,
      button: all.filter((m) => m.type === 'button').length,
      disabled: all.filter((m) => m.status !== 'enable').length,
    }
  }, [treeData])

  const displayTree = useMemo(() => filterTree(treeData, keyword), [treeData, keyword])

  const openModal = (nextMode: ModalMode, parent: MenuItem | null, baseValues: Partial<MenuFormValues>) => {
    setMode(nextMode)
    setEditing(null)
    setParentMenu(parent)
    form.resetFields()
    form.setFieldsValue({ type: 'menu', status: 'enable', hidden: 0, externalLink: 0, keepAlive: 0, ...baseValues })
    setModalOpen(true)
  }

  const handleCreateRoot = () => {
    if (!selectedAppId) {
      message.warning('请先选择所属应用')
      return
    }
    openModal('createRoot', null, { parentID: 0 })
  }

  const handleCreateChild = (parent: MenuItem) => openModal('createChild', parent, { parentID: parent.menuID })

  const handleEdit = (menu: MenuItem) => {
    setMode('edit')
    setEditing(menu)
    setParentMenu(null)
    form.resetFields()
    form.setFieldsValue({
      parentID: menu.parentID,
      name: menu.name,
      code: menu.code,
      path: menu.path,
      icon: menu.icon,
      sort: menu.sort,
      type: menu.type,
      component: menu.component,
      redirect: menu.redirect,
      permission: menu.permission,
      status: menu.status,
      hidden: menu.hidden,
      externalLink: menu.externalLink,
      keepAlive: menu.keepAlive,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    if (!selectedAppId) {
      message.warning('请先选择所属应用')
      return
    }
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      // 新建/编辑始终以列表页选中的应用为准
      if (mode === 'edit' && editing) {
        await updateMenu({ menuID: editing.menuID, appId: selectedAppId, ...values })
        message.success('修改成功')
      } else {
        await createMenu({ appId: selectedAppId, ...values })
        message.success('创建成功')
      }
      setModalOpen(false)
      void fetchData()
    } catch {
      /* 校验或请求失败，拦截器已提示 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleDelete = async (menu: MenuItem) => {
    try {
      await deleteMenu(menu.menuID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  // 上级菜单选项：编辑时排除自身及其子孙，避免形成环
  const parentTreeOptions = useMemo(() => {
    const excluded = new Set<number>()
    if (editing) {
      const collect = (m: MenuItem) => {
        excluded.add(m.menuID)
        m.children?.forEach(collect)
      }
      collect(editing)
    }
    type ParentOption = { value: number; title: string; children?: ParentOption[] }
    const build = (items: MenuTreeNode[]): ParentOption[] =>
      items
        .filter((n) => !excluded.has(n.menu.menuID))
        .map((n) => ({
          value: n.menu.menuID,
          title: `${n.menu.name}（${n.menu.code}）`,
          children: n.children ? build(n.children) : undefined,
        }))
    return [{ value: 0, title: '根菜单', children: build(treeData) }]
  }, [treeData, editing])

  const renderTitle = (node: DataNode) => {
    const menu = (node as MenuTreeNode).menu
    const meta = TYPE_META[menu.type] || { label: menu.type || '菜单', icon: <UnorderedListOutlined />, color: brand.textSecondary }
    return (
      <span className="menu-node">
        <span className="menu-node-main">
          <span style={{ color: meta.color, fontSize: 14 }}>{meta.icon}</span>
          <span style={{ fontWeight: 500 }}>{menu.name}</span>
          <span className="menu-node-code">{menu.code}</span>
          <Tag color={meta.color === '#f59e0b' ? 'orange' : meta.color === '#7a5af8' ? 'purple' : 'blue'} style={{ marginInlineEnd: 4 }}>
            {meta.label}
          </Tag>
          {menu.hidden === 1 && <Tag style={{ marginInlineEnd: 4 }}>隐藏</Tag>}
          <Tag color={menu.status === 'enable' ? 'success' : 'default'} style={{ marginInlineEnd: 0 }}>
            {menu.status === 'enable' ? '启用' : '停用'}
          </Tag>
        </span>
        <span className="menu-node-actions">
          <Button type="link" size="small" onClick={() => handleCreateChild(menu)}>
            新增子菜单
          </Button>
          <Button type="link" size="small" onClick={() => handleEdit(menu)}>
            编辑
          </Button>
          <Popconfirm title={`确认删除「${menu.name}」？其子菜单将一并删除`} onConfirm={() => void handleDelete(menu)}>
            <Button type="link" size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </span>
      </span>
    )
  }

  const modalTitle =
    mode === 'edit'
      ? `编辑菜单 - ${editing?.name || ''}`
      : mode === 'createChild'
        ? `新增子菜单 - ${parentMenu?.name || ''}`
        : '新建根菜单'

  return (
    <PageContainer
      title="菜单管理"
      description="按应用维度维护菜单树与权限标识"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreateRoot}>
            新建根菜单
          </Button>
        </Space>
      }
    >
      <style>{`
        .menu-app-bar {
          display: flex;
          align-items: center;
          justify-content: space-between;
          flex-wrap: wrap;
          gap: 12px;
        }
        .menu-app-label {
          color: ${brand.textSecondary};
          font-size: 13px;
        }
        .menu-app-stats {
          font-size: 13px;
          color: ${brand.textSecondary};
        }
        .menu-app-stats b {
          color: ${brand.text};
          font-size: 16px;
          margin-left: 4px;
        }
        .menu-app-fixed {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 10px 14px;
          border-radius: 8px;
          background: ${brand.gradientSoft};
          font-size: 13px;
        }
        .menu-node {
          display: inline-flex;
          align-items: center;
          justify-content: space-between;
          width: 100%;
          padding-right: 8px;
        }
        .menu-node-main {
          display: inline-flex;
          align-items: center;
          gap: 8px;
        }
        .menu-node-code {
          font-size: 12px;
          color: ${brand.textSecondary};
          font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
        }
        .menu-node-actions {
          opacity: 0;
          transition: opacity 0.15s ease;
          display: inline-flex;
          gap: 2px;
        }
        .ant-tree-treenode:hover .menu-node-actions {
          opacity: 1;
        }
      `}</style>

      {/* 应用切换器 + 菜单概览 */}
      <div className="menu-app-bar">
        <Space size={12}>
          <span className="menu-app-label">所属应用</span>
          <Select
            showSearch
            optionFilterProp="label"
            loading={appLoading}
            placeholder="选择应用"
            style={{ width: 300 }}
            value={selectedAppId}
            onChange={handleAppChange}
            options={apps.map((a) => ({
              value: a.appId,
              label: `${a.name}（${a.code}）`,
            }))}
          />
          {selectedApp && <Tag color="blue">{selectedApp.code}</Tag>}
        </Space>
        <Space size={20} className="menu-app-stats">
          <span>
            菜单总数 <b>{stats.total}</b>
          </span>
          <span>
            目录 <b>{stats.directory}</b>
          </span>
          <span>
            菜单 <b>{stats.menu}</b>
          </span>
          <span>
            按钮 <b>{stats.button}</b>
          </span>
          <span>
            停用 <b>{stats.disabled}</b>
          </span>
        </Space>
      </div>

      <Divider style={{ margin: '12px 0 16px' }} />

      {/* 搜索与展开控制 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <Input
          allowClear
          prefix={<SearchOutlined style={{ color: brand.textSecondary }} />}
          placeholder="按名称 / 编码 / 路径过滤"
          style={{ width: 280 }}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
        <Space>
          <Button size="small" onClick={() => setExpandedKeys(collectKeys(treeData.map((n) => n.menu)))}>
            展开全部
          </Button>
          <Button size="small" onClick={() => setExpandedKeys([])}>
            收起全部
          </Button>
        </Space>
      </div>

      <Spin spinning={treeLoading}>
        {!selectedAppId ? (
          <Empty description="请先选择所属应用" style={{ padding: '48px 0' }} />
        ) : treeData.length === 0 ? (
          <Empty description="该应用暂无菜单，点击右上角「新建根菜单」开始配置" style={{ padding: '48px 0' }} />
        ) : (
          <div style={{ maxHeight: 620, overflow: 'auto', border: '1px solid #f0f0f0', borderRadius: 10, padding: '8px 12px' }}>
            <Tree
              blockNode
              showLine
              treeData={displayTree}
              expandedKeys={keyword ? collectKeys(displayTree.map((n) => n.menu)) : expandedKeys}
              onExpand={(keys) => setExpandedKeys(keys as number[])}
              titleRender={renderTitle}
            />
          </div>
        )}
      </Spin>

      <Modal
        title={modalTitle}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={680}
        styles={{ body: { maxHeight: 580, overflowY: 'auto', paddingTop: 8 } }}
      >
        <Form form={form} layout="vertical">
          {/* 所属应用：以列表页选中为准，只读展示 */}
          <div className="menu-app-fixed">
            <AppstoreOutlined style={{ color: brand.primary }} />
            <span>菜单将归属应用：</span>
            <b>{selectedApp ? `${selectedApp.name}（${selectedApp.code}）` : '-'}</b>
            <span style={{ color: brand.textSecondary, fontSize: 12 }}>（以列表页选中的应用为准，不可修改）</span>
          </div>

          <Divider orientation="left" plain style={{ margin: '16px 0 8px', fontSize: 13 }}>
            基本信息
          </Divider>
          <Form.Item name="parentID" label="上级菜单">
            <TreeSelect
              treeData={parentTreeOptions}
              treeDefaultExpandAll
              placeholder="根菜单"
              allowClear
              showSearch
              treeNodeFilterProp="title"
              style={{ width: '100%' }}
            />
          </Form.Item>
          <div style={{ display: 'flex', gap: 16 }}>
            <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入菜单名称' }]} style={{ flex: 1 }}>
              <Input placeholder="菜单显示名称" />
            </Form.Item>
            <Form.Item name="code" label="编码" rules={[{ required: true, message: '请输入菜单编码' }]} style={{ flex: 1 }}>
              <Input placeholder="唯一编码，如 user:list" />
            </Form.Item>
          </div>
          <div style={{ display: 'flex', gap: 16 }}>
            <Form.Item name="type" label="类型" style={{ flex: 1 }}>
              <Select
                options={[
                  { value: 'directory', label: '目录' },
                  { value: 'menu', label: '菜单' },
                  { value: 'button', label: '按钮' },
                ]}
              />
            </Form.Item>
            <Form.Item name="status" label="状态" style={{ flex: 1 }}>
              <Select
                options={[
                  { value: 'enable', label: '启用' },
                  { value: 'disable', label: '停用' },
                ]}
              />
            </Form.Item>
            <Form.Item name="sort" label="排序" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="默认 0" />
            </Form.Item>
          </div>

          <Divider orientation="left" plain style={{ margin: '16px 0 8px', fontSize: 13 }}>
            路由配置
          </Divider>
          <div style={{ display: 'flex', gap: 16 }}>
            <Form.Item name="path" label="路径" style={{ flex: 1 }}>
              <Input placeholder="如 /user/list" />
            </Form.Item>
            <Form.Item name="component" label="组件" style={{ flex: 1 }}>
              <Input placeholder="如 pages/user/list" />
            </Form.Item>
          </div>
          <div style={{ display: 'flex', gap: 16 }}>
            <Form.Item name="redirect" label="重定向" style={{ flex: 1 }}>
              <Input placeholder="选填" />
            </Form.Item>
            <Form.Item name="icon" label="图标" style={{ flex: 1 }}>
              <Input placeholder="如 UserOutlined" />
            </Form.Item>
          </div>

          <Divider orientation="left" plain style={{ margin: '16px 0 8px', fontSize: 13 }}>
            权限与展示
          </Divider>
          <Form.Item name="permission" label="权限标识">
            <Input placeholder="如 iam:user:create" />
          </Form.Item>
          <div style={{ display: 'flex', gap: 32 }}>
            <Form.Item name="hidden" label="隐藏" valuePropName="checked" getValueFromEvent={(c: boolean) => (c ? 1 : 0)}>
              <Switch checkedChildren="是" unCheckedChildren="否" />
            </Form.Item>
            <Form.Item name="externalLink" label="外链" valuePropName="checked" getValueFromEvent={(c: boolean) => (c ? 1 : 0)}>
              <Switch checkedChildren="是" unCheckedChildren="否" />
            </Form.Item>
            <Form.Item name="keepAlive" label="缓存" valuePropName="checked" getValueFromEvent={(c: boolean) => (c ? 1 : 0)}>
              <Switch checkedChildren="是" unCheckedChildren="否" />
            </Form.Item>
          </div>
        </Form>
      </Modal>
    </PageContainer>
  )
}
