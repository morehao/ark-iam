import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Select, message } from 'antd'
import { tokens } from '@ark-iam/ui'
import type { TenantAppItem, TenantRoleItem, TenantUserRoleItem } from '@ark-iam/types'
import { getMachineUserRoles, updateMachineUserRoles } from '../api/machineUser'
import { getTenantApps } from '../api/menu'
import { getTenantRolePageList } from '../api/role'
import { getTenantUserRoles, updateTenantUserRoles } from '../api/user'

/** 系统/未归属应用组（role.app_id 为空串） */
const SYS_APP_ID = ''
const SYS_APP_NAME = '系统角色'

export interface RoleAssignEditorProps {
  kind: 'user' | 'machine'
  subjectID: string
  /** 保存成功后回调（父级刷新列表/摘要等） */
  onSaved?: () => void
}

interface AppOption {
  appID: string
  appName: string
}

// 应用选项 = 租户订阅应用 ∪ 当前已持有角色所在应用 ∪ 系统角色组（存在空 appID 角色时）。
function buildAppOptions(apps: TenantAppItem[], roles: TenantUserRoleItem[]): AppOption[] {
  const byId = new Map<string, string>()
  for (const a of apps) {
    byId.set(a.appID, a.name)
  }
  for (const r of roles) {
    const key = r.appID || SYS_APP_ID
    const name = r.appName || (r.appID === SYS_APP_ID ? SYS_APP_NAME : '')
    if (!byId.has(key)) {
      byId.set(key, name || `未归属应用（${key}）`)
    }
  }
  return Array.from(byId, ([appID, appName]) => ({ appID, appName }))
}

/**
 * 按应用授权编辑器（用户 / 服务账号共用）：
 * 角色从属于应用（role.app_id），故授权以「主体 × 应用」为粒度 —— 先选目标应用，
 * 再勾选该应用下的角色；保存（PUT）仅全量替换该应用的角色，不影响其它应用。
 */
export default function RoleAssignEditor({ kind, subjectID, onSaved }: RoleAssignEditorProps) {
  const [apps, setApps] = useState<TenantAppItem[]>([])
  const [roles, setRoles] = useState<TenantUserRoleItem[]>([]) // 当前已分配（全应用）
  const [loading, setLoading] = useState(true)
  const [appID, setAppID] = useState<string>()
  const [options, setOptions] = useState<TenantRoleItem[]>([]) // 选中应用下的可授角色
  const [optionsLoading, setOptionsLoading] = useState(false)
  const [checked, setChecked] = useState<string[]>([])
  const [saving, setSaving] = useState(false)

  const loadApps = useCallback(async (): Promise<TenantAppItem[]> => {
    const resp = await getTenantApps().catch(() => null)
    return resp?.list || []
  }, [])

  const loadRoles = useCallback(async (): Promise<TenantUserRoleItem[]> => {
    const resp = kind === 'user' ? await getTenantUserRoles(subjectID) : await getMachineUserRoles(subjectID)
    return resp?.list || []
  }, [kind, subjectID])

  useEffect(() => {
    let alive = true
    setLoading(true)
    Promise.all([loadApps(), loadRoles()])
      .then(([appList, roleList]) => {
        if (!alive) return
        setApps(appList)
        setRoles(roleList)
        // 默认选中：优先「当前持有角色的应用」，否则第一个订阅应用
        const counts = new Map<string, number>()
        roleList.forEach((r) => counts.set(r.appID || SYS_APP_ID, (counts.get(r.appID || SYS_APP_ID) || 0) + 1))
        const ordered = buildAppOptions(appList, roleList)
        setAppID(ordered.find((o) => (counts.get(o.appID) || 0) > 0)?.appID ?? ordered[0]?.appID)
      })
      .catch(() => {})
      .finally(() => {
        if (alive) setLoading(false)
      })
    return () => {
      alive = false
    }
  }, [loadApps, loadRoles])

  const appOptions = useMemo(() => buildAppOptions(apps, roles), [apps, roles])

  const fetchRoleOptions = useCallback(async (target: string): Promise<TenantRoleItem[]> => {
    if (target === SYS_APP_ID) {
      // 空串无法作为 appID 过滤参数（=不过滤全部），全量拉取后客户端过滤出未归属角色
      const all = await getTenantRolePageList({ page: 1, pageSize: 100 })
      return (all?.list || []).filter((r) => !r.appID)
    }
    const resp = await getTenantRolePageList({ appID: target, page: 1, pageSize: 100 })
    return resp?.list || []
  }, [])

  useEffect(() => {
    if (appID === undefined) {
      setOptions([])
      setChecked([])
      return
    }
    let alive = true
    setOptionsLoading(true)
    fetchRoleOptions(appID)
      .then((list) => {
        if (alive) setOptions(list)
      })
      .catch(() => {})
      .finally(() => {
        if (alive) setOptionsLoading(false)
      })
    setChecked(roles.filter((r) => r.appID === appID).map((r) => r.roleID))
    return () => {
      alive = false
    }
  }, [appID, fetchRoleOptions, roles])

  const save = async () => {
    if (appID === undefined) {
      message.warning('请先选择目标应用')
      return
    }
    setSaving(true)
    try {
      if (kind === 'user') {
        await updateTenantUserRoles(subjectID, appID, checked)
      } else {
        await updateMachineUserRoles(subjectID, appID, checked)
      }
      message.success('角色已更新')
      setRoles(await loadRoles())
      onSaved?.()
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <div style={{ color: tokens.textPlaceholder, textAlign: 'center', padding: 24 }}>加载中...</div>
  }

  const currentAppRoles = appID === undefined ? [] : roles.filter((r) => r.appID === appID)
  const currentAppLabel = appOptions.find((o) => o.appID === appID)?.appName || appID || SYS_APP_NAME

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div>
        <div style={{ color: tokens.textPlaceholder, fontSize: 12, marginBottom: 4 }}>目标应用（角色从属于应用，按应用逐个授权）</div>
        <Select
          placeholder="选择目标应用"
          style={{ width: '100%' }}
          value={appID}
          onChange={setAppID}
          options={appOptions.map((o) => {
            const count = roles.filter((r) => r.appID === o.appID).length
            return { value: o.appID, label: `${o.appName}${count > 0 ? `（已分配 ${count}）` : ''}` }
          })}
        />
      </div>
      {appID !== undefined ? (
        <>
          <div>
            <div style={{ color: tokens.textPlaceholder, fontSize: 12, marginBottom: 4 }}>角色（保存后全量替换「{currentAppLabel}」的授权）</div>
            <Select
              mode="multiple"
              allowClear
              showSearch
              optionFilterProp="label"
              loading={optionsLoading}
              placeholder="选择角色"
              style={{ width: '100%' }}
              value={checked}
              onChange={setChecked}
              options={options.map((r) => ({ label: `${r.name}（${r.code}）`, value: r.roleID }))}
            />
          </div>
          <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>
            {kind === 'machine' && '仅可授予普通角色，系统管理角色不可授予服务账号；'}
            当前「{currentAppLabel}」已分配：{currentAppRoles.map((r) => r.name).join('、') || '无'}。保存仅替换该应用，不影响其它应用。
          </div>
          <Button type="primary" loading={saving} onClick={() => void save()} style={{ alignSelf: 'flex-start' }}>
            保存角色
          </Button>
        </>
      ) : (
        <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>暂无可用应用，请确认租户已订阅应用。</div>
      )}
    </div>
  )
}
