import { useCallback, useEffect, useState } from 'react'
import { Table, Button, Space, Input, InputNumber, Modal, Form, Select, Switch, message } from 'antd'
import { MinusCircleOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import {
  actionColumn,
  CODE_COL_WIDTH,
  idColumn,
  NAME_COL_WIDTH,
  nameColumn,
  PageContainer,
  RemoteSelect,
  SourceTag,
  STATUS_COL_WIDTH,
  EnableTag,
  tableScrollX,
  TAG_COL_WIDTH,
  textColumn,
  timeColumn,
  tokens,
} from '@ark-iam/ui'
import {
  createOAuthClient,
  deleteOAuthClient,
  getApplicationPageList,
  getOAuthClientDetail,
  getOAuthClientPageList,
  updateOAuthClient,
} from '@ark-iam/api'
import type { ClientAuthTimeClaimPolicy, ClientPKCEPolicy, GrantType, OAuthClientItem, TokenEndpointAuthMethod } from '@ark-iam/types'
import { useNavigate } from 'react-router-dom'
import { oauthClientDetailPath } from '../../routes'

// 客户端编码（= OIDC client_id）规则，与后端 model.ClientCodePattern 同口径：
// 小写字母开头，仅含小写字母与下划线（禁数字与连字符）。前端与后端各校验一份，改一处必须同步另一处。
const CLIENT_CODE_PATTERN = /^[a-z][a-z_]*$/

// 授权类型/响应类型/Scope 选项与后端 model.GrantType、oidcop 允许的 scope 同口径。
const GRANT_TYPE_OPTIONS = [
  { value: 'authorization_code', label: 'authorization_code' },
  { value: 'refresh_token', label: 'refresh_token' },
  { value: 'client_credentials', label: 'client_credentials' },
]

const RESPONSE_TYPE_OPTIONS = [
  { value: 'code', label: 'code' },
  { value: 'id_token', label: 'id_token' },
  { value: 'token', label: 'token' },
]

// Scopes 用 tags 模式：既可从下拉选择，也可直接输入自定义 scope（自定义值必须登记在 defaultScopes 才会被放行）。
const SCOPE_OPTIONS = [
  { value: 'openid', label: 'openid' },
  { value: 'profile', label: 'profile' },
  { value: 'email', label: 'email' },
  { value: 'phone', label: 'phone' },
]

// 新建缺省协议参数，与后端 application_client 的列默认值同口径（授权类型/响应类型/Scopes/TTL）。
// PKCE 默认关闭以对齐列默认值：公开客户端（none）与 RustFS 这类 S256 接入方需在表单里显式开启。
// requirePKCE/requireAuthTime 已为字符串枚举 enable/disable（P3），不能再出现 boolean/0-1。
const CREATE_DEFAULTS = {
  tokenEndpointAuthMethod: 'client_secret_basic' as TokenEndpointAuthMethod,
  grantTypes: ['authorization_code'] as GrantType[],
  responseTypes: ['code'],
  defaultScopes: ['openid', 'profile', 'email'],
  requirePKCE: 'disable' as ClientPKCEPolicy,
  requireAuthTime: 'disable' as ClientAuthTimeClaimPolicy,
  accessTokenTTL: 900,
  refreshTokenTTL: 2592000,
}

/**
 * 协议开关（requirePKCE/requireAuthTime）是字符串枚举 enable/disable：Switch 需要 boolean，
 * 回显用 getValueProps 把 'enable' 显式转成 checked，提交用 getValueFromEvent 转回字符串枚举。
 */
const ENABLE_FLAG_FORM_PROPS = {
  valuePropName: 'checked' as const,
  getValueProps: (value: unknown) => ({ checked: value === 'enable' }),
  getValueFromEvent: (checked: boolean) => (checked ? 'enable' : 'disable'),
}

/** 字符串数组字段清洗：丢掉 Form.List 的空行与首尾空白，避免把空串写进 redirect_uris 等 JSON 列 */
function cleanStringList(list?: (string | undefined)[]): string[] {
  return (list || []).map((v) => (v || '').trim()).filter((v) => v !== '')
}

/**
 * 字符串数组字段（回调地址 / 登出回调地址 / CORS 白名单共用）：多行输入 + 增删。
 * Form.List 的 name 在父级 Form 内是相对路径，子 Form.Item 直接用它即可。
 */
function StringListField(props: { name: string; label: string; placeholder: string; tooltip?: string }) {
  const { name, label, placeholder, tooltip } = props
  return (
    <Form.Item label={label} tooltip={tooltip}>
      <Form.List name={name}>
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: '100%' }} size={8}>
            {fields.map(({ key, name: fieldName }) => (
              <div key={key} style={{ display: 'flex', gap: 8, alignItems: 'baseline' }}>
                <Form.Item name={fieldName} noStyle>
                  <Input placeholder={placeholder} />
                </Form.Item>
                <Button
                  type="text"
                  danger
                  icon={<MinusCircleOutlined />}
                  aria-label={`删除${label}`}
                  onClick={() => remove(fieldName)}
                />
              </div>
            ))}
            <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add('')}>
              添加{label}
            </Button>
          </Space>
        )}
      </Form.List>
    </Form.Item>
  )
}

export default function OAuthClientList() {
  const navigate = useNavigate()
  const [data, setData] = useState<OAuthClientItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<OAuthClientItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)
  // 内置控制台客户端：编码（= client_id）是网关 aud 白名单与前端构建期默认值的取值来源，保持只读
  const builtinClient = editing?.source === 'builtin'

  // 应用可增长，所属应用下拉走服务端搜索（RemoteSelect），不再一次性只取前 100 条
  const fetchAppOptions = useCallback(async (search: string) => {
    const resp = await getApplicationPageList({ page: 1, pageSize: 50, name: search || undefined })
    return (resp?.list || []).map((a) => ({ value: a.appID, label: a.name }))
  }, [])

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getOAuthClientPageList({ page, pageSize, name: keyword })
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
    form.setFieldsValue(CREATE_DEFAULTS)
    setModalOpen(true)
  }

  /**
   * 编辑态先回显列表行自带的摘要字段（名称/编码/状态/认证方式），再拉详情补全协议参数。
   * 列表接口不回传回调地址、Scopes、PKCE、TTL 等；若只回显行数据，提交会把它们当成空值全量覆盖掉。
   */
  const handleEdit = (record: OAuthClientItem) => {
    setEditing(record)
    form.resetFields()
    form.setFieldsValue({
      code: record.code,
      name: record.name,
      status: record.status,
      tokenEndpointAuthMethod: record.tokenEndpointAuthMethod,
    })
    setModalOpen(true)
    void (async () => {
      try {
        const detail = await getOAuthClientDetail(record.applicationClientID)
        // 只补列表接口没有的协议参数：不回写 编码/名称/状态/认证方式——
        // 详情若返回慢于用户输入，回写会把刚改的内容冲掉，而这四项列表行已是最新值
        form.setFieldsValue({
          redirectURIs: detail.redirectURIs || [],
          postLogoutRedirectURIs: detail.postLogoutRedirectURIs || [],
          backChannelLogoutURI: detail.backChannelLogoutURI || '',
          grantTypes: detail.grantTypes || [],
          responseTypes: detail.responseTypes || [],
          defaultScopes: detail.defaultScopes || [],
          allowedOrigins: detail.allowedOrigins || [],
          requirePKCE: detail.requirePKCE,
          requireAuthTime: detail.requireAuthTime,
          accessTokenTTL: detail.accessTokenTTL,
          refreshTokenTTL: detail.refreshTokenTTL,
        })
      } catch {
        /* 拦截器已提示：详情拉取失败时保留摘要回显，不阻塞其它字段编辑 */
      }
    })()
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      // 列表字段清洗为字符串数组：编辑接口是全量覆盖，缺字段会把这几个 JSON 列清空
      const payload = {
        ...values,
        redirectURIs: cleanStringList(values.redirectURIs),
        postLogoutRedirectURIs: cleanStringList(values.postLogoutRedirectURIs),
        allowedOrigins: cleanStringList(values.allowedOrigins),
        defaultScopes: cleanStringList(values.defaultScopes),
        backChannelLogoutURI: String(values.backChannelLogoutURI || '').trim(),
      }
      setSubmitLoading(true)
      if (editing) {
        await updateOAuthClient({ applicationClientID: editing.applicationClientID, ...payload })
        message.success('修改成功')
      } else {
        await createOAuthClient(payload)
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

  const handleDelete = async (record: OAuthClientItem) => {
    try {
      await deleteOAuthClient(record.applicationClientID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<OAuthClientItem> = [
    idColumn<OAuthClientItem>({ dataIndex: 'applicationClientID' }),
    nameColumn<OAuthClientItem>({
      title: '名称',
      dataIndex: 'name',
      // 名称即详情入口（列规范 R2）；路由 path 取自 routes.ts，与 App.tsx 注册保持一致
      onClick: (r) => navigate(oauthClientDetailPath(r.applicationClientID)),
    }),
    // 列名与后端字段名同构（DTO/DB 都是 code）：客户端编码 = OIDC client_id
    textColumn<OAuthClientItem>({ title: '客户端编码', dataIndex: 'code', width: CODE_COL_WIDTH, monospace: true }),
    textColumn<OAuthClientItem>({ title: '所属应用', dataIndex: 'appName', width: NAME_COL_WIDTH }),
    { title: '来源', dataIndex: 'source', key: 'source', width: TAG_COL_WIDTH, render: (v: string) => <SourceTag value={v} /> },
    { title: '状态', dataIndex: 'status', key: 'status', width: STATUS_COL_WIDTH, render: (v: string) => <EnableTag value={v} /> },
    timeColumn<OAuthClientItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<OAuthClientItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<OAuthClientItem>({
      max: 2,
      actions: (r) => [
        { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
        {
          key: 'delete',
          label: '删除',
          danger: true,
          // 内置客户端（source=builtin，网关 aud 白名单与前端构建期 client_id 的来源）禁删：置灰保留展示；
          // 后端 svcapplicationclient.Delete 以 ApplicationClientBuiltInErr 兜底。
          disabled: r.source === 'builtin',
          confirm: '确认删除该客户端？',
          onClick: () => void handleDelete(r),
        },
      ],
    }),
  ]

  return (
    <PageContainer
      title="OAuth 客户端"
      description="管理应用接入的 OIDC/OAuth 客户端与密钥"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建客户端
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按名称搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => { setKeyword(v); setPage(1) }}
        />
      </div>
      <Table<OAuthClientItem>
        rowKey="applicationClientID"
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
        title={editing ? '编辑客户端' : '新建客户端'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={760}
      >
        <Form form={form} layout="vertical">
          {editing && (
            <div style={{ marginBottom: 16, color: tokens.textSecondary, fontSize: 12 }}>
              保存会按当前表单全量覆盖协议参数（回调地址 / 授权类型 / Scopes / 有效期等），请确认各项后再提交。
            </div>
          )}
          {!editing && (
            <Form.Item name="appID" label="所属应用" rules={[{ required: true, message: '请选择所属应用' }]}>
              <RemoteSelect placeholder="选择所属应用（输入名称搜索）" fetchOptions={fetchAppOptions} />
            </Form.Item>
          )}
          <Form.Item
            name="code"
            label="客户端编码"
            tooltip={
              builtinClient
                ? '内置控制台的 OIDC 身份，由平台版本定义，不可修改'
                : '即 OIDC 的 client_id。改动后使用它的应用需同步自己的 client_id 配置，且其已签发令牌会失效'
            }
            rules={[
              { required: true, message: '请输入客户端编码' },
              { pattern: CLIENT_CODE_PATTERN, message: '以小写字母开头，仅含小写字母与下划线' },
            ]}
          >
            <Input placeholder="唯一编码，如 iam_client" disabled={builtinClient} />
          </Form.Item>
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
          >
            <Input placeholder="客户端名称" />
          </Form.Item>
          {editing && (
            <Form.Item label="来源">
              <SourceTag value={editing.source} />
              <span style={{ marginLeft: 8, color: tokens.textSecondary }}>来源不可修改</span>
            </Form.Item>
          )}
          <Form.Item
            name="tokenEndpointAuthMethod"
            label="令牌端点认证方式"
            tooltip={
              builtinClient
                ? '内置控制台是浏览器公共客户端，必须保持 none + 强制 PKCE：它的代码会下发给每个用户，无法保密客户端密钥（RFC 6749 §10.1 / RFC 10017 §6.3.3.1）；改成机密客户端会当场锁死该控制台'
                : 'none = 公共客户端（浏览器 / 移动端，不发密钥，必须配强制 PKCE）；client_secret_basic / client_secret_post = 机密客户端（服务端，可签发密钥）'
            }
            rules={[{ required: true, message: '请选择认证方式' }]}
          >
            <Select
              disabled={builtinClient}
              options={[
                { value: 'client_secret_basic', label: 'client_secret_basic' },
                { value: 'client_secret_post', label: 'client_secret_post' },
                { value: 'none', label: 'none' },
              ]}
            />
          </Form.Item>

          <StringListField
            name="redirectURIs"
            label="回调地址"
            placeholder="https://app.example.com/auth/callback"
            tooltip="授权码回调地址，必须与接入方请求里的 redirect_uri 完全一致（含 scheme、端口与路径），否则授权请求会被拒绝"
          />
          <StringListField
            name="postLogoutRedirectURIs"
            label="登出回调地址"
            placeholder="https://app.example.com/login"
            tooltip="RP-Initiated Logout 的 post_logout_redirect_uri 白名单"
          />
          <Form.Item
            name="backChannelLogoutURI"
            label="后端登出地址"
            tooltip="OP 直接回调接入方的 back-channel logout 端点（服务端到服务端，不受浏览器登出影响）"
          >
            <Input placeholder="https://app.example.com/oidc/bc-logout" />
          </Form.Item>

          <Form.Item
            name="grantTypes"
            label="授权类型"
            rules={[{ required: true, message: '请选择授权类型' }]}
          >
            <Select mode="multiple" options={GRANT_TYPE_OPTIONS} placeholder="authorization_code / refresh_token" />
          </Form.Item>
          <Form.Item
            name="responseTypes"
            label="响应类型"
            rules={[{ required: true, message: '请选择响应类型' }]}
          >
            <Select mode="multiple" options={RESPONSE_TYPE_OPTIONS} placeholder="code" />
          </Form.Item>
          <Form.Item
            name="defaultScopes"
            label="默认 Scopes"
            tooltip="授权请求未显式带 scope 时下发的权限范围；也可输入自定义 scope（自定义值需登记在此才会被放行）"
            rules={[{ required: true, message: '请选择或输入 scope' }]}
          >
            <Select mode="tags" options={SCOPE_OPTIONS} placeholder="openid / profile / email" />
          </Form.Item>
          <StringListField
            name="allowedOrigins"
            label="CORS 白名单"
            placeholder="https://app.example.com"
            tooltip="允许浏览器跨域直连 token/userinfo 的 Origin 白名单，按精确 origin 匹配"
          />

          <Space size={24} align="start">
            <Form.Item
              name="requirePKCE"
              label="强制 PKCE"
              {...ENABLE_FLAG_FORM_PROPS}
              tooltip={
                builtinClient
                  ? '内置控制台必须保持开启：公共客户端没有密钥，PKCE 是唯一的授权码绑定手段，关掉等于放弃防降级'
                  : '开启后授权请求必须携带 code_challenge；公开客户端（none）与使用 S256 PKCE 的接入方（如 RustFS 控制台）需要开启'
              }
            >
              <Switch disabled={builtinClient} />
            </Form.Item>
            <Form.Item
              name="requireAuthTime"
              label="需要 auth_time"
              {...ENABLE_FLAG_FORM_PROPS}
              tooltip="要求 id_token 带 auth_time 声明（max_age / 强制重新认证场景）"
            >
              <Switch />
            </Form.Item>
          </Space>

          <Space size={24} align="start">
            <Form.Item
              name="accessTokenTTL"
              label="AccessToken 有效期（秒）"
              rules={[{ required: true, message: '请输入有效期' }]}
            >
              <InputNumber min={60} precision={0} />
            </Form.Item>
            <Form.Item
              name="refreshTokenTTL"
              label="RefreshToken 有效期（秒）"
              rules={[{ required: true, message: '请输入有效期' }]}
            >
              <InputNumber min={60} precision={0} />
            </Form.Item>
          </Space>

          {editing && (
            <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}>
              <Select
                options={[
                  { value: 'enable', label: '启用' },
                  { value: 'disable', label: '停用' },
                ]}
              />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </PageContainer>
  )
}
