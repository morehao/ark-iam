import { useCallback, useEffect, useState } from 'react'
import { Button, Space, Table } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { EllipsisCell, IDCell, timeColumn, tokens } from '@ark-iam/ui'
import type { TenantUserLoginLogItem } from '@ark-iam/types'
import { getTenantUserLoginLogs } from '../api/user'

export interface UserLoginLogTabProps {
  userID: string
}

/** 用户登录日志（只读）：租户与用户双重过滤由后端保证。 */
export default function UserLoginLogTab({ userID }: UserLoginLogTabProps) {
  const [data, setData] = useState<TenantUserLoginLogItem[]>([])
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTenantUserLoginLogs(userID)
      setData(resp?.list || [])
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [userID])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const columns: ColumnsType<TenantUserLoginLogItem> = [
    { title: 'ID', dataIndex: 'userLoginLogID', key: 'userLoginLogID', width: 140, render: (v: string) => <IDCell value={v} /> },
    { title: '登录IP', dataIndex: 'loginIP', key: 'loginIP', width: 140, render: (v: string) => v || '-' },
    { title: 'UserAgent', dataIndex: 'userAgent', key: 'userAgent', render: (v: string) => <EllipsisCell value={v} /> },
    timeColumn<TenantUserLoginLogItem>({ title: '登录时间', dataIndex: 'loginTime', relative: true }),
  ]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Space>
        <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
          刷新
        </Button>
      </Space>

      <Table<TenantUserLoginLogItem>
        rowKey="userLoginLogID"
        loading={loading}
        columns={columns}
        dataSource={data}
        pagination={false}
        scroll={{ x: 660 }}
      />

      {!loading && data.length === 0 ? (
        <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>暂无登录日志。</div>
      ) : null}
    </div>
  )
}
