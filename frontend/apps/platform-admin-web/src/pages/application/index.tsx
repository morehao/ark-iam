import { useCallback, useEffect, useState } from 'react'
import { Button, Descriptions, Drawer, Form, Input, InputNumber, message, Modal, Select, Space, Switch, Table } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { actionColumn, CODE_COL_WIDTH, fmtTime, IDCell, idColumn, nameColumn, PageContainer, SourceTag, STATUS_COL_WIDTH, EnableTag, tableScrollX, TAG_COL_WIDTH, textColumn, timeColumn, tokens } from '@ark-iam/ui'
import { createApplication, deleteApplication, getApplicationDetail, getApplicationPageList, updateApplication } from '@ark-iam/api'
import type { ApplicationItem, ApplicationRoleTemplateItem } from '@ark-iam/types'

// 应用编码规则（与后端 model.AppCodePattern 同口径）：以小写字母开头，仅含小写字母、数字与下划线。
const APP_CODE_PATTERN = /^[a-z][a-z0-9_]*$/
// 角色模板的契约值编码规则（与后端 model.RoleCodePattern 同口径，跨语言无法共享正则，改一处必须同步另一处）。
const ROLE_CODE_PATTERN = /^[a-z][a-z0-9_]*$/

/**
 * 计算本次更新相对原模板被移除的编码。
 * 编码（而非名称）是各租户物化角色的定位值，故撤下判定只看编码。
 */
function removedTemplateCodes(
  before?: ApplicationRoleTemplateItem[],
  after?: ApplicationRoleTemplateItem[],
): string[] {
  if (!before?.length) return []
  const next = new Set((after ?? []).map((item) => item.code))
  return before.filter((item) => !next.has(item.code)).map((item) => item.code)
}

/**
 * 撤下已下发角色的二次确认：模板移除会在**各租户**删除该角色并级联删除成员授权与菜单授权，
 * 不可从控制台恢复（只能重新加回模板并要求租户重新授权），因此必须显式确认。
 */
function confirmTemplateWithdraw(codes: string[]): Promise<boolean> {
  return new Promise((resolve) => {
    Modal.confirm({
      title: '确认撤下已下发的角色？',
      content: `将从所有已开通该应用的租户中删除角色：${codes.join('、')}，其成员授权与菜单授权一并撤销且不可恢复。`,
      okText: '确认撤下',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: () => resolve(true),
      onCancel: () => resolve(false),
    })
  })
}

export default function ApplicationList() {
  const [data, setData] = useState<ApplicationItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<ApplicationItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)
  // 内置应用（source=builtin）：编码是控制台菜单入口的定位值（platform_admin/tenant_admin），保持只读
  const builtinApp = editing?.source === 'builtin'

  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<ApplicationItem | null>(null)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getApplicationPageList({ page, pageSize, name: keyword })
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
    // 两个入口策略默认关闭（与后端列默认值 false 一致）；不开则对应自助通道的整体拒绝
    form.setFieldsValue({ sort: 0, allowPersonCreateTenant: false, allowJoinByInvite: false, roleTemplate: [] })
    setModalOpen(true)
  }

  const handleEdit = (record: ApplicationItem) => {
    setEditing(record)
    form.setFieldsValue({
      code: record.code,
      name: record.name,
      status: record.status,
      description: record.description,
      logoUrl: record.logoUrl,
      homepageUrl: record.homepageUrl,
      sort: record.sort,
      // 后端列为可空 *bool，未配置（NULL）时语义等同关闭，此处归一成 false 交给 Switch
      allowPersonCreateTenant: !!record.allowPersonCreateTenant,
      allowJoinByInvite: !!record.allowJoinByInvite,
      roleTemplate: record.roleTemplate ?? [],
    })
    setModalOpen(true)
  }

  const handleOpenDetail = async (record: ApplicationItem) => {
    setDetail(record)
    setDetailOpen(true)
    try {
      const resp = await getApplicationDetail(record.appID)
      setDetail(resp)
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      // 模板是单一事实源：从模板移除的编码会在各已开通租户里删掉该角色（连同成员与菜单授权），
      // 属不可逆撤权，提交前必须二次确认（后端同步逻辑见 pkg/core/tenant.SyncAppRoleTemplateToTenants）
      if (editing) {
        const removed = removedTemplateCodes(editing.roleTemplate, values.roleTemplate)
        if (removed.length > 0) {
          const confirmed = await confirmTemplateWithdraw(removed)
          if (!confirmed) return
        }
      }
      setSubmitLoading(true)
      if (editing) {
        await updateApplication({ appID: editing.appID, ...values })
        message.success('修改成功')
      } else {
        await createApplication(values)
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

  const handleDelete = async (record: ApplicationItem) => {
    try {
      await deleteApplication(record.appID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<ApplicationItem> = [
    idColumn<ApplicationItem>({ dataIndex: 'appID' }),
    nameColumn<ApplicationItem>({
      title: '应用名',
      dataIndex: 'name',
      onClick: (r) => void handleOpenDetail(r),
    }),
    textColumn<ApplicationItem>({ title: '编码', dataIndex: 'code', width: CODE_COL_WIDTH, monospace: true }),
    { title: '来源', dataIndex: 'source', key: 'source', width: TAG_COL_WIDTH, render: (v: string) => <SourceTag value={v} /> },
    { title: '状态', dataIndex: 'status', key: 'status', width: STATUS_COL_WIDTH, render: (v: string) => <EnableTag value={v} /> },
    timeColumn<ApplicationItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<ApplicationItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<ApplicationItem>({
      max: 2,
      actions: (r) => [
        { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
        {
          key: 'delete',
          label: '删除',
          danger: true,
          // 内置应用（source=builtin，平台管理后台/租户管理后台）由平台版本交付，禁删：置灰保留展示；
          // 后端 svcapplication.Delete 以 ApplicationBuiltInErr 兜底。
          disabled: r.source === 'builtin',
          confirm: '确认删除该应用？',
          onClick: () => void handleDelete(r),
        },
      ],
    }),
  ]

  return (
    <PageContainer
      title="应用管理"
      description="接入平台的应用"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建应用
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按应用名搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
      </div>
      <Table<ApplicationItem>
        rowKey="appID"
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
        title={editing ? '编辑应用' : '新建应用'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="code"
            label="应用编码"
            tooltip={
              builtinApp
                ? '内置应用的编码由平台版本定义，控制台菜单入口按它定位，不可修改'
                : '应用编码可修改；菜单/订阅/角色按应用 ID 关联，改编码不影响它们'
            }
            rules={[
              { required: true, message: '请输入应用编码' },
              { pattern: APP_CODE_PATTERN, message: '以小写字母开头，仅含小写字母、数字与下划线' },
            ]}
          >
            <Input placeholder="唯一编码，如 iam_web" disabled={builtinApp} />
          </Form.Item>
          <Form.Item
            name="name"
            label="应用名称"
            rules={[{ required: true, message: '请输入应用名称' }]}
          >
            <Input placeholder="应用名称" />
          </Form.Item>
          {editing && (
            <Form.Item label="来源">
              <SourceTag value={editing.source} />
              <span style={{ marginLeft: 8, color: tokens.textSecondary }}>来源不可修改</span>
            </Form.Item>
          )}
          {editing && (
            <Form.Item name="status" label="状态">
              <Select
                options={[
                  { value: 'enable', label: '启用' },
                  { value: 'disable', label: '停用' },
                ]}
              />
            </Form.Item>
          )}
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="选填" />
          </Form.Item>
          <Form.Item name="logoUrl" label="Logo 地址">
            <Input placeholder="https://... 选填" />
          </Form.Item>
          <Form.Item name="homepageUrl" label="首页地址">
            <Input placeholder="https://... 选填" />
          </Form.Item>
          <Form.Item name="sort" label="排序">
            <InputNumber style={{ width: '100%' }} placeholder="数字越小越靠前" />
          </Form.Item>
          <Form.Item
            name="allowPersonCreateTenant"
            label="个人自助创建租户"
            valuePropName="checked"
            tooltip="开启后，经本应用登录的零租户用户可自助注册并开通自己的租户（通道 A），注册人成为该租户拥有者"
          >
            <Switch checkedChildren="允许" unCheckedChildren="禁止" />
          </Form.Item>
          <Form.Item
            name="allowJoinByInvite"
            label="允许邀请加入租户"
            valuePropName="checked"
            tooltip="开启后，经本应用登录的用户可凭邀请码加入已有租户（通道 B），加入者恒为普通成员；关闭时 joinTenant 一律拒绝"
          >
            <Switch checkedChildren="允许" unCheckedChildren="禁止" />
          </Form.Item>
          {/*
            应用角色模板：本应用对外提供的跨系统授权契约值（= OIDC ID token 的 groups 取值）。
            下游按「claim_prefix + 编码」认策略名，而策略在下游是全局命名实体、全租户共用一条，
            因此契约值只能在这里定义一次，租户侧只能授权、不能造值。
          */}
          <Form.List name="roleTemplate">
            {(fields, { add, remove }) => (
              <Form.Item
                label="角色模板"
                tooltip="本应用对外的契约角色：编码即下游策略名取值（下游按「claim_prefix + 编码」认策略），名称会作为租户侧角色的展示名。开通本应用的租户会自动获得这些角色；保存时会同步到所有已开通租户——从模板中移除的编码会连同其在各租户的角色与授权一并撤下。"
              >
                {fields.map((field) => (
                  <Space key={field.key} align="baseline" style={{ display: 'flex', marginBottom: 8 }}>
                    <Form.Item
                      name={[field.name, 'code']}
                      rules={[
                        { required: true, message: '请输入编码' },
                        { pattern: ROLE_CODE_PATTERN, message: '以小写字母开头，仅含小写字母、数字与下划线' },
                      ]}
                      style={{ marginBottom: 0 }}
                    >
                      <Input placeholder="编码，如 storage_admin" style={{ width: 220 }} />
                    </Form.Item>
                    <Form.Item
                      name={[field.name, 'name']}
                      rules={[{ required: true, message: '请输入名称' }]}
                      style={{ marginBottom: 0 }}
                    >
                      <Input placeholder="名称，如 存储管理员" style={{ width: 220 }} />
                    </Form.Item>
                    <Button type="link" danger onClick={() => remove(field.name)}>
                      移除
                    </Button>
                  </Space>
                ))}
                <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({ code: '', name: '' })}>
                  添加契约角色
                </Button>
              </Form.Item>
            )}
          </Form.List>
        </Form>
      </Modal>

      <Drawer
        title={detail ? `应用详情 - ${detail.name || ''}` : '应用详情'}
        width={560}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
      >
        {detail && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="应用ID"><IDCell value={detail.appID} /></Descriptions.Item>
            <Descriptions.Item label="编码">{detail.code || '-'}</Descriptions.Item>
            <Descriptions.Item label="名称">{detail.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="来源">
              <SourceTag value={detail.source} />
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <EnableTag value={detail.status} />
            </Descriptions.Item>
            <Descriptions.Item label="描述">{detail.description || '-'}</Descriptions.Item>
            <Descriptions.Item label="Logo 地址">{detail.logoUrl || '-'}</Descriptions.Item>
            <Descriptions.Item label="首页地址">{detail.homepageUrl || '-'}</Descriptions.Item>
            <Descriptions.Item label="排序">{detail.sort ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{fmtTime(detail.createdAt)}</Descriptions.Item>
            <Descriptions.Item label="个人自助创建租户">{detail.allowPersonCreateTenant ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="允许邀请加入租户">{detail.allowJoinByInvite ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="角色模板">
              {detail.roleTemplate?.length
                ? detail.roleTemplate.map((item) => `${item.name}（${item.code}）`).join('、')
                : '-'}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </PageContainer>
  )
}
