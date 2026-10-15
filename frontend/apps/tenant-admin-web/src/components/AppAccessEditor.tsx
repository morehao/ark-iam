import { useCallback, useEffect, useMemo, useState } from 'react'
import { Alert, Button, Checkbox, Modal, Tag, message } from 'antd'
import { tokens } from '@ark-iam/ui'
import type { TenantUserApplicationItem } from '@ark-iam/types'
import { getTenantUserApplications, updateTenantUserApplications } from '../api/user'

export interface AppAccessEditorProps {
  userID: string
  /** 被编辑成员姓名（确认文案里指名道姓，避免改错人） */
  userName?: string
  /** 保存成功后回调（父级刷新列表/摘要等） */
  onSaved?: () => void
}

/**
 * 应用准入编辑器（用户 × 应用）。
 *
 * 准入是「用户能否进入某应用」的唯一事实源（订阅是它的上界）：
 * - 可勾选集 = 本租户已订阅且启用的应用（未订阅的应用不出现，因为订阅不满足时准入必然无效）；
 * - 已勾选 = 该成员已准入且启用的应用；
 * - 保存走 `PUT /tenant/users/{userID}/applications` **全量替换**（传空数组 = 清空全部准入）。
 *
 * 撤销准入会在同一事务内**级联撤销该应用下的角色绑定**（不变式 I2），
 * 因此对 `roleCount > 0` 的应用在提交前弹二次确认，明确告知将同时失去多少个角色。
 */
export default function AppAccessEditor({ userID, userName, onSaved }: AppAccessEditorProps) {
  const [items, setItems] = useState<TenantUserApplicationItem[]>([])
  const [loading, setLoading] = useState(true)
  const [checked, setChecked] = useState<string[]>([])
  const [saving, setSaving] = useState(false)

  const load = useCallback(async (): Promise<TenantUserApplicationItem[]> => {
    const resp = await getTenantUserApplications(userID).catch(() => null)
    return resp?.list || []
  }, [userID])

  useEffect(() => {
    let alive = true
    setLoading(true)
    load()
      .then((list) => {
        if (!alive) return
        setItems(list)
        setChecked(list.filter((it) => it.granted && it.subscribed && it.appStatus === 'enable').map((it) => it.appID))
      })
      .catch(() => {})
      .finally(() => {
        if (alive) setLoading(false)
      })
    return () => {
      alive = false
    }
  }, [load])

  // 可勾选集合：只有「租户已订阅且应用自身启用」的应用才谈得上准入
  const selectable = useMemo(() => items.filter((it) => it.subscribed && it.appStatus === 'enable'), [items])

  // 订阅已失效但准入行仍在的应用（不可勾选、不可见）。全量替换时必须把它们原样带回：
  // 不变式 I3 —— 取消订阅不级联删除准入，遗漏即等于"顺手撤销"，会造成重新订阅后访问不恢复。
  const retainedGrantIDs = useMemo(
    () => items.filter((it) => it.granted && !(it.subscribed && it.appStatus === 'enable')).map((it) => it.appID),
    [items],
  )

  // 本次将被撤销、且会级联撤销角色绑定的应用（二次确认文案的数据来源）
  const revokingWithRoles = useMemo(
    () => selectable.filter((it) => it.granted && !checked.includes(it.appID) && it.roleCount > 0),
    [selectable, checked],
  )

  const save = async () => {
    setSaving(true)
    try {
      // 全量替换：本次勾选 ∪ 订阅失效但保留的准入行
      await updateTenantUserApplications(userID, Array.from(new Set([...checked, ...retainedGrantIDs])))
      message.success('应用准入已更新')
      setItems(await load())
      onSaved?.()
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSaving(false)
    }
  }

  const onSaveClick = () => {
    if (revokingWithRoles.length === 0) {
      void save()
      return
    }
    const detail = revokingWithRoles.map((it) => `${it.name}（${it.roleCount} 个角色）`).join('、')
    Modal.confirm({
      title: '撤销应用准入将同时撤销角色',
      content: `取消「${userName || '该成员'}」对 ${detail} 的准入后，系统会同时撤销其在该应用下的角色绑定，成员将无法进入该应用。确认继续？`,
      okText: '确认撤销',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: () => save(),
    })
  }

  if (loading) {
    return <div style={{ color: tokens.textPlaceholder, textAlign: 'center', padding: 24 }}>加载中...</div>
  }

  if (selectable.length === 0) {
    return <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>本租户暂无已订阅且启用的应用，请先在「应用订阅」中开通后再分配准入。</div>
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Alert
        type="info"
        showIcon
        message="准入是进入应用的前提"
        description="未勾选的应用，该成员即使被授予角色也无法登录（角色授予会自动补齐准入）。保存为全量替换，取消勾选会在同一事务内撤销该应用下的角色绑定。"
      />
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {selectable.map((it) => (
          <Checkbox
            key={it.appID}
            checked={checked.includes(it.appID)}
            onChange={(e) =>
              setChecked((prev) => (e.target.checked ? [...prev, it.appID] : prev.filter((id) => id !== it.appID)))
            }
          >
            <span style={{ marginRight: 8 }}>{it.name}</span>
            {it.roleCount > 0 && <Tag color="blue">{it.roleCount} 个角色</Tag>}
            {it.source === 'provision' && <Tag>引导写入</Tag>}
            {it.source === 'role' && <Tag>角色授予</Tag>}
          </Checkbox>
        ))}
      </div>
      {/* 订阅已失效但准入行仍在的应用：如实呈现"准入被订阅挡住"的真相（重新订阅即恢复） */}
      {items.filter((it) => !(it.subscribed && it.appStatus === 'enable')).length > 0 && (
        <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>
          另有 {items.filter((it) => !(it.subscribed && it.appStatus === 'enable')).length} 个应用当前不可用（未订阅 / 订阅已停用 /
          应用已停用），其准入行会保留：重新订阅或恢复应用后自动恢复。
        </div>
      )}
      <Button type="primary" loading={saving} onClick={onSaveClick} style={{ alignSelf: 'flex-start' }}>
        保存准入
      </Button>
    </div>
  )
}
