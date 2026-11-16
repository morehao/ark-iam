import { useCallback, useEffect, useMemo, useState } from 'react'
import { Alert, Button, Card, Form, Input, InputNumber, Space, Switch, message } from 'antd'
import { ReloadOutlined, SaveOutlined } from '@ant-design/icons'
import { PageContainer } from '@ark-iam/ui'
import { getSettingPageList, updateSetting, createSetting } from '@ark-iam/api'
import type { SettingItem } from '@ark-iam/types'

/** 掩码值：后端对加密项回显此值；提交该值表示「不修改」。 */
const MASKED_VALUE = '******'

/**
 * 由分组配置项数组拼成表单值的键值对。
 * 加密项（掩码）原样保留 MASKED_VALUE，保存时后端按掩码识别为「不改」。
 */
function itemsToValues(items: SettingItem[]): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const it of items) {
    let v: unknown = it.value
    if (it.valueType === 'bool') v = it.value === 'true'
    else if (it.valueType === 'int' || it.valueType === 'float') v = it.value === '' ? undefined : Number(it.value)
    out[it.key] = v
  }
  return out
}

/** Form 值里某个 key 转成提交用的字符串。 */
function valueToStr(v: unknown, type: string): string {
  if (type === 'bool') return v ? 'true' : 'false'
  return v === undefined || v === null ? '' : String(v)
}

interface GroupField {
  key: string
  label: string
  type: 'bool' | 'int' | 'string' | 'secret'
  help?: string
}

interface GroupDef {
  group: string
  title: string
  description: string
  fields: GroupField[]
}

const GROUPS: GroupDef[] = [
  {
    group: 'notify',
    title: '通知渠道（临时密码投递）',
    description: '账号初始/重置临时密码的投递渠道。总闸关闭时回退为「创建成功后回显一次明文」。',
    fields: [
      { key: 'enabled', label: '总闸', type: 'bool', help: '关闭=不发；开启=按渠道投递' },
      { key: 'requireDelivery', label: '投递失败报错', type: 'bool', help: '开启=投递失败报错（需重置重试）；关闭=回退回显' },
      { key: 'mail.enabled', label: '邮件渠道', type: 'bool' },
      { key: 'mail.host', label: 'SMTP Host', type: 'string' },
      { key: 'mail.port', label: 'SMTP 端口', type: 'int' },
      { key: 'mail.username', label: 'SMTP 用户名', type: 'string' },
      { key: 'mail.password', label: 'SMTP 密码', type: 'secret', help: '留空/掩码表示不修改' },
      { key: 'mail.from', label: '发件人', type: 'string' },
      { key: 'mail.tls', label: '启用 TLS', type: 'bool' },
      { key: 'sms.enabled', label: '短信渠道', type: 'bool' },
      { key: 'sms.provider', label: '短信供应商', type: 'string', help: 'tencent；空=不启用' },
      { key: 'sms.accessKeyID', label: '短信 AccessKeyID', type: 'string' },
      { key: 'sms.accessKeySecret', label: '短信 AccessKeySecret', type: 'secret', help: '留空/掩码表示不修改' },
      { key: 'sms.signName', label: '短信签名', type: 'string' },
      { key: 'sms.sdkAppId', label: '短信 SdkAppId', type: 'string' },
      { key: 'sms.templateCode', label: '短信模板 ID', type: 'string' },
      { key: 'sms.region', label: '短信地域', type: 'string' },
    ],
  },
  {
    group: 'security.login',
    title: '登录安全（失败锁定 + 限流）',
    description: '登录失败锁定阈值与登录接口频率限流参数，保存后下一请求生效。',
    fields: [
      { key: 'maxFailures', label: '失败次数阈值', type: 'int', help: '默认 5' },
      { key: 'windowSec', label: '失败计数窗口（秒）', type: 'int', help: '默认 300' },
      { key: 'lockSec', label: '锁定时长（秒）', type: 'int', help: '默认 900' },
      { key: 'ratePerMinute', label: '限流：每分钟请求数', type: 'int', help: '默认 30' },
      { key: 'burst', label: '限流：突发容量', type: 'int', help: '默认 10' },
    ],
  },
]

export default function SettingPage() {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [items, setItems] = useState<SettingItem[]>([])
  const [form] = Form.useForm()

  const itemsByKey = useMemo(() => {
    const m = new Map<string, SettingItem>()
    for (const it of items) m.set(`${it.group}:${it.key}`, it)
    return m
  }, [items])

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getSettingPageList({ page: 1, pageSize: 1000 })
      const list = resp?.list || []
      setItems(list)
      // 所有分组拼成一个扁平表单值（key 用 group 前缀避免跨组分 key 冲突）
      const values: Record<string, unknown> = {}
      for (const g of GROUPS) {
        const groupItems = list.filter((it) => it.group === g.group)
        const vs = itemsToValues(groupItems)
        for (const [k, v] of Object.entries(vs)) values[`${g.group}::${k}`] = v
      }
      form.setFieldsValue(values)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [form])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      for (const g of GROUPS) {
        for (const f of g.fields) {
          const formKey = `${g.group}::${f.key}`
          const raw = values[formKey]
          const type = f.type === 'secret' ? 'string' : f.type
          const existing = itemsByKey.get(`${g.group}:${f.key}`)
          if (!existing) {
            // 该字段尚无配置行：仅当有非空值时才创建
            const sv = valueToStr(raw, type)
            if (sv === '' || raw === undefined || raw === null) continue
            await createSetting({
              group: g.group,
              key: f.key,
              valueType: type,
              value: sv,
              encrypted: f.type === 'secret',
            })
          } else {
            // 已有行：secret 掩码/空 -> 不传 value；其它按当前值更新
            const sv = valueToStr(raw, type)
            const updateBody: { value?: string; encrypted?: boolean } = {}
            if (f.type === 'secret') {
              if (raw !== undefined && raw !== null && String(raw) !== '' && String(raw) !== MASKED_VALUE) {
                updateBody.value = String(raw)
                updateBody.encrypted = true
              }
            } else {
              updateBody.value = sv
            }
            if (Object.keys(updateBody).length > 0) {
              await updateSetting(existing.id, updateBody)
            }
          }
        }
      }
      message.success('保存成功')
      void fetchData()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSaving(false)
    }
  }

  const renderField = (g: GroupDef, f: GroupField) => {
    const formKey = `${g.group}::${f.key}`
    let control: React.ReactNode
    if (f.type === 'bool') {
      control = <Switch />
    } else if (f.type === 'int') {
      control = <InputNumber style={{ width: 220 }} min={0} />
    } else if (f.type === 'secret') {
      control = <Input.Password style={{ width: 320 }} placeholder="留空表示不修改" autoComplete="new-password" />
    } else {
      control = <Input style={{ width: 320 }} />
    }
    return (
      <Form.Item key={formKey} name={formKey} label={f.label} extra={f.help} valuePropName={f.type === 'bool' ? 'checked' : 'value'}>
        {control}
      </Form.Item>
    )
  }

  return (
    <PageContainer
      title="系统设置"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<SaveOutlined />} onClick={() => void handleSave()} loading={saving}>
            保存
          </Button>
        </Space>
      }
    >
      <Alert type="info" showIcon message="这些参数保存后即时生效，无需重启服务。敏感字段以掩码展示，留空或保持掩码表示不修改。" style={{ marginBottom: 16 }} />
      <Form form={form} layout="vertical" disabled={loading}>
        {GROUPS.map((g) => (
          <Card key={g.group} title={g.title} style={{ marginBottom: 16 }}>
            <div style={{ color: '#8c8c8c', marginBottom: 12 }}>{g.description}</div>
            {g.fields.map((f) => renderField(g, f))}
          </Card>
        ))}
      </Form>
    </PageContainer>
  )
}
