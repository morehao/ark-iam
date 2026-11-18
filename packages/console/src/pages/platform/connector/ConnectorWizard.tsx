import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  Button,
  Card,
  Col,
  Divider,
  Form,
  Input,
  Row,
  Select,
  Space,
  Steps,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd'
import { CheckCircleFilled } from '@ant-design/icons'
import { tokens } from '@ark-iam/ui'
import { createConnector, getConnectorFactories, getTenantPageList, updateConnector } from '@ark-iam/api'
import type {
  ConnectorClaimMapping,
  ConnectorConfig,
  ConnectorCreateReq,
  ConnectorDomainPolicy,
  ConnectorFactoryItem,
  ConnectorItem,
  TenantItem,
} from '@ark-iam/types'
import { CAPABILITY_LABEL, PROVIDER_COLOR } from './constants'

/** enable/disable 字符串枚举 <-> antd Switch 布尔值（真值判断禁止）。 */
const FLAG_PROPS = {
  valuePropName: 'checked' as const,
  getValueProps: (value: unknown) => ({ checked: value === 'enable' }),
  getValueFromEvent: (checked: boolean) => (checked ? 'enable' : 'disable'),
}

/** 表单内部值形态（提交时再组装成后端 ConnectorConfig 结构）。 */
interface ConnectorFormValues {
  name: string
  displayName: string
  status: string
  defaultTenantID?: string
  issuer?: string
  /** 协议级 tenant（部分 IdP 的租户标识），与 defaultTenantID 语义不同。 */
  tenant?: string
  authUrl?: string
  tokenUrl?: string
  userInfoUrl?: string
  clientId?: string
  clientSecret?: string
  redirectUri?: string
  scopes?: string[]
  claimMapping?: ConnectorClaimMapping
  allowedDomains?: string[]
  blockedDomains?: string[]
  allowAutoCreateUser?: string
  allowAccountLink?: string
  syncProfile?: string
  enableTokenStorage?: string
}

interface Props {
  open: boolean
  editing: ConnectorItem | null
  onClose: () => void
  onSaved: () => void
}

const STEPS = ['选择提供商', '基本信息', '协议配置', '声明映射', '域策略与开关']

/** 把表单值 + 所选工厂组装为后端创建/更新请求体（config 为结构化对象，非扁平字段）。 */
function buildPayload(factory: ConnectorFactoryItem, v: ConnectorFormValues): ConnectorCreateReq {
  const config: ConnectorConfig = {
    protocol: factory.protocol,
    provider: factory.provider,
    issuer: v.issuer,
    authUrl: v.authUrl,
    tokenUrl: v.tokenUrl,
    userInfoUrl: v.userInfoUrl,
    clientID: v.clientId,
    clientSecret: v.clientSecret || undefined,
    redirectUri: v.redirectUri,
    scopes: v.scopes,
    tenant: v.tenant,
  }
  return {
    name: v.name,
    displayName: v.displayName,
    protocol: factory.protocol,
    provider: factory.provider,
    status: (v.status || 'enable') as ConnectorCreateReq['status'],
    // 默认落租户：仅在自动创建用户=enable 时提交，否则置空。
    defaultTenantID: v.allowAutoCreateUser === 'enable' ? v.defaultTenantID : undefined,
    allowAutoCreateUser: (v.allowAutoCreateUser || 'disable') as ConnectorCreateReq['allowAutoCreateUser'],
    allowAccountLink: (v.allowAccountLink || 'disable') as ConnectorCreateReq['allowAccountLink'],
    syncProfile: (v.syncProfile || 'disable') as ConnectorCreateReq['syncProfile'],
    enableTokenStorage: (v.enableTokenStorage || 'disable') as ConnectorCreateReq['enableTokenStorage'],
    config,
    claimMapping: v.claimMapping ?? {},
    domainPolicy: {
      allowedDomains: v.allowedDomains ?? [],
      blockedDomains: v.blockedDomains ?? [],
    } satisfies ConnectorDomainPolicy,
  }
}

export default function ConnectorWizard({ open, editing, onClose, onSaved }: Props) {
  const [current, setCurrent] = useState(0)
  const [form] = Form.useForm<ConnectorFormValues>()
  const [factories, setFactories] = useState<ConnectorFactoryItem[]>([])
  const [factoryID, setFactoryID] = useState<string>('')
  const [tenants, setTenants] = useState<TenantItem[]>([])
  const [submitLoading, setSubmitLoading] = useState(false)
  // 受控跟踪「自动创建用户」开关，用于动态显示/校验「默认落租户」。
  const [allowAutoCreate, setAllowAutoCreate] = useState<string>('disable')

  const factory = useMemo(
    () => factories.find((f) => f.factoryID === factoryID) ?? null,
    [factories, factoryID],
  )

  // 加载工厂列表 + 平台租户列表（一次）
  useEffect(() => {
    if (!open) return
    getConnectorFactories()
      .then((resp) => setFactories(resp?.list || []))
      .catch(() => {
        /* 拦截器已提示 */
      })
    getTenantPageList({ page: 1, pageSize: 200 })
      .then((resp) => setTenants(resp?.list || []))
      .catch(() => {
        /* 拦截器已提示 */
      })
  }, [open])

  // 打开时初始化表单（**仅依赖 open**：不能依赖 factories，否则工厂异步到位会二次 resetFields，
  // 把用户在「基本信息」步已填的 name/displayName 清空——E2E 曾因此建出空名连接器）。
  useEffect(() => {
    if (!open) return
    if (editing) {
      setCurrent(1)
      const c = editing.config ?? {}
      setAllowAutoCreate(editing.allowAutoCreateUser)
      form.setFieldsValue({
        name: editing.name,
        displayName: editing.displayName,
        status: editing.status,
        defaultTenantID: editing.defaultTenantID || undefined,
        issuer: c.issuer,
        tenant: c.tenant,
        authUrl: c.authUrl,
        tokenUrl: c.tokenUrl,
        userInfoUrl: c.userInfoUrl,
        clientId: c.clientID,
        // clientSecret 后端返回脱敏值（******），编辑时留空表示不修改
        clientSecret: '',
        redirectUri: c.redirectUri,
        scopes: c.scopes,
        claimMapping: editing.claimMapping,
        allowedDomains: editing.domainPolicy?.allowedDomains,
        blockedDomains: editing.domainPolicy?.blockedDomains,
        allowAutoCreateUser: editing.allowAutoCreateUser,
        allowAccountLink: editing.allowAccountLink,
        syncProfile: editing.syncProfile,
        enableTokenStorage: editing.enableTokenStorage,
      })
    } else {
      setFactoryID('')
      setCurrent(0)
      setAllowAutoCreate('disable')
      form.resetFields()
      form.setFieldsValue({
        status: 'enable',
        allowAutoCreateUser: 'disable',
        allowAccountLink: 'disable',
        syncProfile: 'disable',
        enableTokenStorage: 'disable',
      })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  // 工厂到位后（仅编辑态）按 provider 回填 factoryID（回填 config 不依赖工厂，故无需 reset 表单）
  useEffect(() => {
    if (!open || !editing || factories.length === 0) return
    const f = factories.find((x) => x.provider === editing.provider) ?? null
    setFactoryID(f?.factoryID ?? '')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, editing, factories])

  const pickFactory = (f: ConnectorFactoryItem) => {
    setFactoryID(f.factoryID)
    form.setFieldsValue({ scopes: f.defaultScopes })
    setCurrent(1)
  }

  const next = async () => {
    const fieldSets: (string | string[])[][] = [
      [],
      ['name', 'displayName', 'status'],
      factory?.configSchema?.required ?? [],
      [],
      [],
    ]
    try {
      await form.validateFields(fieldSets[current] as never)
      setCurrent((c) => c + 1)
    } catch {
      /* 校验失败停留 */
    }
  }

  const submit = async () => {
    if (!factory) return
    try {
      const values = await form.validateFields()
      const payload = buildPayload(factory, values)
      setSubmitLoading(true)
      if (editing) {
        // 编辑且未填新 secret：buildPayload 已将其置 undefined，不覆盖原值
        await updateConnector(editing.connectorID, payload)
        message.success('修改成功')
      } else {
        await createConnector(payload)
        message.success('创建成功')
      }
      onSaved()
    } catch {
      /* 校验/请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const required = factory?.configSchema?.required ?? []
  const isRequired = (k: string) => required.includes(k)

  const renderProtocol = () => {
    if (!factory) return null
    return (
      <>
        <Row gutter={16}>
          {isRequired('issuer') && (
            <Col span={12}>
              <Form.Item name="issuer" label="Issuer（签发者）" rules={[{ required: true, message: '请输入 Issuer' }]}>
                <Input placeholder="https://accounts.google.com" />
              </Form.Item>
            </Col>
          )}
          {isRequired('tenant') && (
            <Col span={12}>
              <Form.Item name="tenant" label="租户标识 Tenant（协议）" rules={[{ required: true, message: '请输入租户标识' }]}>
                <Input placeholder="contoso.onmicrosoft.com" />
              </Form.Item>
            </Col>
          )}
          {isRequired('authUrl') && (
            <Col span={12}>
              <Form.Item name="authUrl" label="授权端点 Auth URL" rules={[{ required: true, message: '请输入授权端点' }]}>
                <Input placeholder="https://github.com/login/oauth/authorize" />
              </Form.Item>
            </Col>
          )}
          {isRequired('tokenUrl') && (
            <Col span={12}>
              <Form.Item name="tokenUrl" label="令牌端点 Token URL" rules={[{ required: true, message: '请输入令牌端点' }]}>
                <Input placeholder="https://github.com/login/oauth/access_token" />
              </Form.Item>
            </Col>
          )}
          {isRequired('userInfoUrl') && (
            <Col span={12}>
              <Form.Item
                name="userInfoUrl"
                label="用户信息端点 UserInfo URL"
                rules={[{ required: true, message: '请输入用户信息端点' }]}
              >
                <Input placeholder="https://api.github.com/user" />
              </Form.Item>
            </Col>
          )}
          <Col span={12}>
            <Form.Item name="clientId" label="Client ID" rules={[{ required: isRequired('clientId'), message: '请输入 Client ID' }]}>
              <Input placeholder="由身份提供商分配" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="clientSecret"
              label="Client Secret"
              rules={[{ required: !editing && isRequired('clientSecret'), message: '请输入 Client Secret' }]}
              tooltip={editing ? '敏感信息不回显；不修改请留空' : '敏感信息，保存后不在详情中明文展示'}
            >
              <Input.Password
                placeholder={editing ? '不修改请留空' : '由身份提供商分配'}
                autoComplete="new-password"
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="redirectUri"
              label="回调地址 Redirect URI"
              rules={[{ required: isRequired('redirectUri'), message: '请输入回调地址' }]}
              tooltip="需与身份提供商后台登记的回调地址完全一致"
            >
              <Input placeholder="https://iam.example.com/oidc/callback" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="scopes" label="Scope 范围" tooltip="由工厂给出的默认值，可按需增减">
              <Select mode="tags" placeholder="选择或输入 scope" />
            </Form.Item>
          </Col>
        </Row>
        <Alert
          type="info"
          showIcon
          message="此步骤字段由所选提供商工厂（configSchema.required）决定，切换提供商时字段集随之变化。"
        />
      </>
    )
  }

  const renderStepContent = (idx: number) => {
    switch (idx) {
      case 0:
        return (
          <Row gutter={16}>
            {factories.map((f) => (
              <Col span={8} key={f.factoryID}>
                <Card
                  hoverable
                  onClick={() => pickFactory(f)}
                  style={{
                    borderColor: factoryID === f.factoryID ? tokens.primary : tokens.border,
                    borderWidth: factoryID === f.factoryID ? 2 : 1,
                  }}
                  styles={{ body: { padding: 16 } }}
                >
                  <Space direction="vertical" size={8} style={{ width: '100%' }}>
                    <Space style={{ justifyContent: 'space-between', width: '100%' }}>
                      <Typography.Text strong>{f.displayName}</Typography.Text>
                      <Tag color={PROVIDER_COLOR[f.provider]}>{f.protocol.toUpperCase()}</Tag>
                    </Space>
                    <div style={{ color: tokens.textSecondary, fontSize: 12 }}>
                      默认 Scope：{f.defaultScopes.join(', ')}
                    </div>
                    <Space wrap size={4}>
                      {f.capabilities.map((cap) => (
                        <Tag key={cap} variant="filled" color="blue">
                          {CAPABILITY_LABEL[cap] ?? cap}
                        </Tag>
                      ))}
                    </Space>
                    <Button
                      type={factoryID === f.factoryID ? 'primary' : 'default'}
                      size="small"
                      icon={factoryID === f.factoryID ? <CheckCircleFilled /> : undefined}
                      onClick={(e) => {
                        e.stopPropagation()
                        pickFactory(f)
                      }}
                    >
                      {factoryID === f.factoryID ? '已选择' : '选择'}
                    </Button>
                  </Space>
                </Card>
              </Col>
            ))}
          </Row>
        )
      case 1:
        return (
          <>
            {factory && (
              <Alert
                style={{ marginBottom: 16 }}
                type="success"
                showIcon
                message={`已选择提供商：${factory.displayName}（${factory.protocol.toUpperCase()}）`}
                action={
                  !editing ? (
                    <Button size="small" type="link" onClick={() => setCurrent(0)}>
                      更换
                    </Button>
                  ) : undefined
                }
              />
            )}
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="name"
                  label="标识名"
                  tooltip="作为该连接器的稳定标识，建议小写字母/数字/连字符"
                  rules={[
                    { required: true, message: '请输入标识名' },
                    { pattern: /^[a-z][a-z0-9-]*$/, message: '以小写字母开头，仅含小写字母、数字与连字符' },
                  ]}
                >
                  <Input placeholder="如 google-workspace" disabled={!!editing} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="displayName" label="显示名称" rules={[{ required: true, message: '请输入显示名称' }]}>
                  <Input placeholder="如 Google Workspace" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="status" label="状态" tooltip="停用后保留配置，但不参与授权与回调">
                  <Select
                    options={[
                      { value: 'enable', label: '启用' },
                      { value: 'disable', label: '停用' },
                    ]}
                  />
                </Form.Item>
              </Col>
            </Row>
          </>
        )
      case 2:
        return renderProtocol()
      case 3:
        return (
          <>
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message="声明映射：把外部 IdP 返回的 claim 名映射到平台标准身份字段。留空则使用提供商默认映射。"
            />
            <Row gutter={16}>
              {(
                [
                  ['subject', 'Subject（唯一用户标识）'],
                  ['email', 'Email（邮箱）'],
                  ['username', 'Username（用户名）'],
                  ['displayName', 'DisplayName（显示名）'],
                  ['avatarUrl', 'AvatarUrl（头像地址）'],
                ] as const
              ).map(([key, label]) => (
                <Col span={12} key={key}>
                  <Form.Item name={['claimMapping', key]} label={label}>
                    <Input placeholder={`外部 claim 名，如 ${key}`} />
                  </Form.Item>
                </Col>
              ))}
            </Row>
          </>
        )
      case 4:
        return (
          <>
            <Typography.Text strong>域策略</Typography.Text>
            <Typography.Paragraph type="secondary" style={{ fontSize: 12, marginTop: 4 }}>
              限制哪些邮箱域可以通过本连接器登录。允许列表非空时仅这些域可登录；拒绝列表优先级更高。
            </Typography.Paragraph>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="allowedDomains" label="允许的邮箱域">
                  <Select mode="tags" placeholder="如 acme.com（留空则不限制）" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="blockedDomains" label="拒绝的邮箱域">
                  <Select mode="tags" placeholder="如 contractor.com" />
                </Form.Item>
              </Col>
            </Row>
            <Divider style={{ margin: '8px 0 16px' }} />
            <Typography.Text strong>策略开关</Typography.Text>
            <Row gutter={16} style={{ marginTop: 12 }}>
              <Col span={12}>
                <Form.Item
                  name="allowAutoCreateUser"
                  label="自动创建用户"
                  {...FLAG_PROPS}
                  tooltip="首次通过本连接器登录、平台无对应用户时，是否自动建档"
                >
                  <Switch checkedChildren="允许" unCheckedChildren="禁止" onChange={(checked) => setAllowAutoCreate(checked ? 'enable' : 'disable')} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="defaultTenantID"
                  label="默认落租户"
                  rules={[
                    {
                      required: allowAutoCreate === 'enable',
                      message: '自动创建用户开启时，必须选择默认落租户',
                    },
                  ]}
                  tooltip="自动创建用户时，用户默认落在该租户下"
                >
                  <Select
                    allowClear
                    showSearch
                    optionFilterProp="label"
                    placeholder="选择默认落租户"
                    options={tenants.map((t) => ({ value: t.tenantID, label: t.name }))}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="allowAccountLink"
                  label="账号关联"
                  {...FLAG_PROPS}
                  tooltip="外部身份是否可与平台已有账号关联绑定"
                >
                  <Switch checkedChildren="允许" unCheckedChildren="禁止" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="syncProfile" label="资料同步" {...FLAG_PROPS} tooltip="登录时同步外部资料到平台用户">
                  <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="enableTokenStorage"
                  label="令牌存储"
                  {...FLAG_PROPS}
                  tooltip="是否存储外部 access/refresh token，供后续调用外部 API"
                >
                  <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                </Form.Item>
              </Col>
            </Row>
          </>
        )
      default:
        return null
    }
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: tokens.bg,
        zIndex: 1000,
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div style={{ background: tokens.cardBg, borderBottom: `1px solid ${tokens.border}`, padding: '16px 24px' }}>
        <Space style={{ justifyContent: 'space-between', width: '100%' }}>
          <Typography.Title level={5} style={{ margin: 0 }}>
            {editing ? `编辑连接器 · ${editing.displayName}` : '新建连接器'}
          </Typography.Title>
          <Button onClick={onClose}>取消</Button>
        </Space>
      </div>

      <div style={{ padding: '20px 24px 0' }}>
        <Steps
          current={current}
          size="small"
          items={STEPS.map((title) => ({ title }))}
          style={{ maxWidth: 900, margin: '0 auto' }}
        />
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
        <Card style={{ maxWidth: 900, margin: '0 auto', minHeight: 380 }} styles={{ body: { padding: 24 } }}>
          <Form form={form} layout="vertical" requiredMark="optional" preserve>
            {/* 五步全部保持挂载、仅隐藏非当前步：避免 antd Form.Item 因条件卸载而丢失已填值
                （曾导致「基本信息/协议配置」填了却没进提交 payload）。 */}
            {STEPS.map((_, idx) => (
              <div key={idx} style={{ display: idx === current ? 'block' : 'none' }}>
                {renderStepContent(idx)}
              </div>
            ))}
          </Form>
        </Card>
      </div>

      <div
        style={{
          background: tokens.cardBg,
          borderTop: `1px solid ${tokens.border}`,
          padding: '12px 24px',
          display: 'flex',
          justifyContent: 'flex-end',
          gap: 8,
        }}
      >
        <Button disabled={current === 0} onClick={() => setCurrent((c) => c - 1)}>
          上一步
        </Button>
        {current < STEPS.length - 1 ? (
          <Button type="primary" onClick={() => void next()}>
            下一步
          </Button>
        ) : (
          <Button type="primary" loading={submitLoading} onClick={() => void submit()}>
            {editing ? '保存修改' : '创建连接器'}
          </Button>
        )}
      </div>
    </div>
  )
}
