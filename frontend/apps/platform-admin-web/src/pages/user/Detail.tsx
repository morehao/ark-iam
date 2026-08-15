import { useCallback, useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Descriptions, Tabs, Table, Button, Spin, Space, Avatar, Tag, Modal, Form, Input, Popconfirm, message, Tree, Empty } from 'antd'
import { ArrowLeftOutlined, PlusOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { DataNode } from 'antd/es/tree'
import {
  assignUserDepartments,
  createUserIdentity,
  deleteUserIdentity,
  getDepartmentTree,
  getUserDepartmentByUser,
  getUserDetail,
  getUserIdentityByUser,
  getUserLoginLogByUser,
} from '@ark-iam/api'
import type { DepartmentItem, UserIdentityItem, UserItem, UserLoginLogItem } from '@ark-iam/types'
import { fmtTime, SuspendedTag } from '../../components/common'
import { brand } from '@ark-iam/ui'

export default function UserDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [user, setUser] = useState<UserItem | null>(null)
  const [loading, setLoading] = useState(false)

  const [identities, setIdentities] = useState<UserIdentityItem[]>([])
  const [loginLogs, setLoginLogs] = useState<UserLoginLogItem[]>([])
  const [deptRel, setDeptRel] = useState<{ departmentID: number; isPrimary: number }[]>([])
  const [deptTree, setDeptTree] = useState<DataNode[]>([])
  const [assignModalOpen, setAssignModalOpen] = useState(false)
  const [checkedDeptKeys, setCheckedDeptKeys] = useState<React.Key[]>([])
  const [identityModalOpen, setIdentityModalOpen] = useState(false)
  const [identityForm] = Form.useForm()

  const userID = Number(id)

  const fetchAll = useCallback(async () => {
    if (!id) return
    setLoading(true)
    try {
      const [u, ids, logs, depts] = await Promise.allSettled([
        getUserDetail(userID),
        getUserIdentityByUser(userID),
        getUserLoginLogByUser(userID),
        getUserDepartmentByUser(userID),
      ])
      if (u.status === 'fulfilled') setUser(u.value)
      if (ids.status === 'fulfilled') setIdentities(ids.value.list || [])
      if (logs.status === 'fulfilled') setLoginLogs(logs.value.list || [])
      if (depts.status === 'fulfilled') setDeptRel(depts.value.list || [])
    } finally {
      setLoading(false)
    }
  }, [id, userID])

  useEffect(() => {
    void fetchAll()
  }, [fetchAll])

  const loadDeptTree = async () => {
    try {
      const resp = await getDepartmentTree()
      const build = (list: DepartmentItem[]): DataNode[] =>
        list.map((d) => ({ key: d.departmentID, title: d.name, children: d.children ? build(d.children) : [] }))
      setDeptTree(build(resp.list || []))
    } catch {
      /* 拦截器已提示 */
    }
  }

  const openAssign = async () => {
    await loadDeptTree()
    setCheckedDeptKeys(deptRel.map((d) => d.departmentID))
    setAssignModalOpen(true)
  }

  const handleAssign = async () => {
    try {
      await assignUserDepartments(userID, checkedDeptKeys.map(Number))
      message.success('分配成功')
      setAssignModalOpen(false)
      void fetchAll()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleAddIdentity = async () => {
    try {
      const values = await identityForm.validateFields()
      await createUserIdentity({ tenantID: user?.tenantID ?? 0, userID: userID, issuer: values.issuer, identityID: values.identityID })
      message.success('绑定成功')
      setIdentityModalOpen(false)
      identityForm.resetFields()
      void fetchAll()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const identityColumns: ColumnsType<UserIdentityItem> = [
    { title: 'ID', dataIndex: 'userIdentityID', key: 'userIdentityID', width: 90 },
    { title: '身份提供商', dataIndex: 'issuer', key: 'issuer' },
    { title: '第三方用户ID', dataIndex: 'identityID', key: 'identityID' },
    { title: '创建时间', key: 'createdAt', width: 170, render: (_, r) => fmtTime((r as UserIdentityItem & { createdAt?: number }).createdAt) },
    {
      title: '操作',
      key: 'action',
      width: 90,
      render: (_, r) => (
        <Popconfirm
          title="确认解绑该身份？"
          onConfirm={async () => {
            await deleteUserIdentity(userID, r.userIdentityID)
            message.success('解绑成功')
            void fetchAll()
          }}
        >
          <Button type="link" size="small" danger>
            解绑
          </Button>
        </Popconfirm>
      ),
    },
  ]

  const logColumns: ColumnsType<UserLoginLogItem> = [
    { title: 'ID', dataIndex: 'userLoginLogID', key: 'userLoginLogID', width: 90 },
    { title: '登录IP', dataIndex: 'loginIP', key: 'loginIP', width: 150 },
    { title: 'UserAgent', dataIndex: 'userAgent', key: 'userAgent', ellipsis: true },
    { title: '登录时间', key: 'loginTime', width: 170, render: (_, r) => fmtTime(r.loginTime) },
  ]

  if (loading && !user) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: 80 }}>
        <Spin size="large" />
      </div>
    )
  }

  if (!user) return null

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/user')}>
          返回列表
        </Button>
      </Space>

      <Card style={{ borderRadius: 12, marginBottom: 16, border: '1px solid #f0f0f0' }} styles={{ body: { padding: 24 } }}>
        <Space size={20} align="center">
          <Avatar size={64} style={{ background: brand.gradient, fontSize: 26 }}>
            {user.name?.charAt(0)?.toUpperCase() || 'U'}
          </Avatar>
          <div>
            <Space size={12} align="center">
              <span style={{ fontSize: 20, fontWeight: 700 }}>{user.name || '-'}</span>
              <SuspendedTag value={user.isSuspended} />
            </Space>
            <div style={{ marginTop: 6, color: brand.textSecondary, fontSize: 13 }}>
              @{user.username || '-'} · {user.primaryEmail || '未绑定邮箱'} · {user.primaryPhone || '未绑定手机'}
            </div>
          </div>
        </Space>
      </Card>

      <Card style={{ borderRadius: 12, border: '1px solid #f0f0f0' }}>
        <Tabs
          items={[
            {
              key: 'info',
              label: '基本信息',
              children: (
                <Descriptions column={2} bordered size="small">
                  <Descriptions.Item label="用户ID">{user.userID}</Descriptions.Item>
                  <Descriptions.Item label="租户ID">{user.tenantID}</Descriptions.Item>
                  <Descriptions.Item label="用户名">{user.username || '-'}</Descriptions.Item>
                  <Descriptions.Item label="姓名">{user.name || '-'}</Descriptions.Item>
                  <Descriptions.Item label="邮箱">{user.primaryEmail || '-'}</Descriptions.Item>
                  <Descriptions.Item label="手机号">{user.primaryPhone || '-'}</Descriptions.Item>
                </Descriptions>
              ),
            },
            {
              key: 'identity',
              label: '第三方身份',
              children: (
                <>
                  <Space style={{ marginBottom: 12 }}>
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => setIdentityModalOpen(true)}>
                      绑定身份
                    </Button>
                  </Space>
                  <Table<UserIdentityItem> rowKey="userIdentityID" columns={identityColumns} dataSource={identities} pagination={false} />
                </>
              ),
            },
            {
              key: 'loginLog',
              label: '登录日志',
              children: (
                <Table<UserLoginLogItem> rowKey="userLoginLogID" columns={logColumns} dataSource={loginLogs} pagination={false} scroll={{ x: 700 }} />
              ),
            },
            {
              key: 'department',
              label: '部门归属',
              children: (
                <>
                  <Space style={{ marginBottom: 12 }}>
                    <Button type="primary" onClick={() => void openAssign()}>
                      分配部门
                    </Button>
                  </Space>
                  {deptRel.length === 0 ? (
                    <Empty description="尚未分配部门" />
                  ) : (
                    <Table
                      rowKey="departmentID"
                      dataSource={deptRel}
                      pagination={false}
                      columns={[
                        { title: '部门ID', dataIndex: 'departmentID', key: 'departmentID' },
                        {
                          title: '是否主部门',
                          dataIndex: 'isPrimary',
                          key: 'isPrimary',
                          render: (v: number) => (v === 1 ? <Tag color="blue">主部门</Tag> : <Tag>普通</Tag>),
                        },
                      ]}
                    />
                  )}
                </>
              ),
            },
          ]}
        />
      </Card>

      <Modal title="分配部门" open={assignModalOpen} onOk={() => void handleAssign()} onCancel={() => setAssignModalOpen(false)} width={420}>
        <Tree
          checkable
          treeData={deptTree}
          checkedKeys={checkedDeptKeys}
          onCheck={(keys) => setCheckedDeptKeys(Array.isArray(keys) ? keys : keys.checked)}
          defaultExpandAll
        />
      </Modal>

      <Modal title="绑定第三方身份" open={identityModalOpen} onOk={() => void handleAddIdentity()} onCancel={() => setIdentityModalOpen(false)} destroyOnClose>
        <Form form={identityForm} layout="vertical">
          <Form.Item name="issuer" label="身份提供商" rules={[{ required: true, message: '请输入身份提供商' }]}>
            <Input placeholder="如 https://accounts.google.com" />
          </Form.Item>
          <Form.Item name="identityID" label="第三方用户ID" rules={[{ required: true, message: '请输入第三方用户ID' }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
