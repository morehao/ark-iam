import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Spin, Switch, Tree, message } from 'antd'
import type { DataNode } from 'antd/es/tree'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import { PageContainer, brand } from '@ark-iam/ui'
import { createMenu, deleteMenu, getMenuTree, updateMenu } from '@ark-iam/api'
import type { MenuItem } from '@ark-iam/types'

interface MenuTreeNode extends DataNode {
  menu: MenuItem
}

interface MenuFormValues {
  parentID: number
  appId?: number
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

export default function MenuList() {
  const [treeData, setTreeData] = useState<MenuTreeNode[]>([])
  const [expandedKeys, setExpandedKeys] = useState<number[]>([])
  const [treeLoading, setTreeLoading] = useState(false)

  const [modalOpen, setModalOpen] = useState(false)
  const [mode, setMode] = useState<ModalMode>('createRoot')
  const [editing, setEditing] = useState<MenuItem | null>(null)
  const [parentMenu, setParentMenu] = useState<MenuItem | null>(null)
  const [form] = Form.useForm<MenuFormValues>()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setTreeLoading(true)
    try {
      const resp = await getMenuTree()
      const list = resp?.list || []
      setTreeData(buildTreeData(list))
      setExpandedKeys(collectKeys(list))
    } catch {
      /* 拦截器已提示 */
    } finally {
      setTreeLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const openModal = (nextMode: ModalMode, parent: MenuItem | null, baseValues: Partial<MenuFormValues>) => {
    setMode(nextMode)
    setEditing(null)
    setParentMenu(parent)
    form.resetFields()
    form.setFieldsValue({ type: 'menu', status: 'enable', hidden: 0, externalLink: 0, keepAlive: 0, ...baseValues })
    setModalOpen(true)
  }

  const handleCreateRoot = () => openModal('createRoot', null, { parentID: 0 })

  const handleCreateChild = (parent: MenuItem) => openModal('createChild', parent, { parentID: parent.menuID })

  const handleEdit = (menu: MenuItem) => {
    setMode('edit')
    setEditing(menu)
    setParentMenu(null)
    form.resetFields()
    form.setFieldsValue({
      parentID: menu.parentID,
      appId: menu.appId,
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
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (mode === 'edit' && editing) {
        await updateMenu({ menuID: editing.menuID, ...values })
        message.success('修改成功')
      } else {
        await createMenu(values)
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

  const renderTitle = (node: DataNode) => {
    const menu = (node as MenuTreeNode).menu
    return (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
        <span>
          <span style={{ fontWeight: 500 }}>{menu.name}</span>
          <span style={{ fontSize: 12, color: brand.textSecondary }}>({menu.code})</span>
        </span>
        <span style={{ display: 'inline-flex', gap: 4 }} onClick={(e) => e.stopPropagation()}>
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
      description="应用菜单与权限标识配置"
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
      <Spin spinning={treeLoading}>
        <div style={{ maxHeight: 620, overflow: 'auto' }}>
          <Tree
            blockNode
            showLine
            treeData={treeData}
            expandedKeys={expandedKeys}
            onExpand={(keys) => setExpandedKeys(keys as number[])}
            titleRender={renderTitle}
          />
        </div>
      </Spin>

      <Modal
        title={modalTitle}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={640}
        styles={{ body: { maxHeight: 560, overflowY: 'auto' } }}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="parentID" label="父菜单 ID" rules={[{ required: true, message: '请输入父菜单 ID' }]}>
            <InputNumber style={{ width: '100%' }} disabled={mode !== 'edit'} placeholder="根菜单为 0" />
          </Form.Item>
          <Form.Item name="appId" label="应用 ID">
            <InputNumber style={{ width: '100%' }} placeholder="选填" />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入菜单名称' }]}>
            <Input placeholder="菜单显示名称" />
          </Form.Item>
          <Form.Item name="code" label="编码" rules={[{ required: true, message: '请输入菜单编码' }]}>
            <Input placeholder="唯一编码，如 user:list" />
          </Form.Item>
          <Form.Item name="path" label="路径">
            <Input placeholder="选填，如 /user/list" />
          </Form.Item>
          <Form.Item name="icon" label="图标">
            <Input placeholder="选填，如 UserOutlined" />
          </Form.Item>
          <Form.Item name="sort" label="排序">
            <InputNumber style={{ width: '100%' }} placeholder="选填，默认 0" />
          </Form.Item>
          <Form.Item name="type" label="类型">
            <Select placeholder="选填">
              <Select.Option value="directory">目录</Select.Option>
              <Select.Option value="menu">菜单</Select.Option>
              <Select.Option value="button">按钮</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="component" label="组件">
            <Input placeholder="选填，如 pages/user/list" />
          </Form.Item>
          <Form.Item name="redirect" label="重定向">
            <Input placeholder="选填" />
          </Form.Item>
          <Form.Item name="permission" label="权限标识">
            <Input placeholder="选填，如 iam:user:create" />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select placeholder="选填">
              <Select.Option value="enable">启用</Select.Option>
              <Select.Option value="disable">停用</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="hidden" label="隐藏" valuePropName="checked" getValueFromEvent={(c: boolean) => (c ? 1 : 0)}>
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>
          <Form.Item name="externalLink" label="外链" valuePropName="checked" getValueFromEvent={(c: boolean) => (c ? 1 : 0)}>
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>
          <Form.Item name="keepAlive" label="缓存" valuePropName="checked" getValueFromEvent={(c: boolean) => (c ? 1 : 0)}>
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
