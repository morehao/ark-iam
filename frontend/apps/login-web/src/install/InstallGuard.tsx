import { useEffect, useState, type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { getInstallStatus } from '../api'
import '../LoginPage.css'

/**
 * 登录入口守卫：GET /install/status 显示 initialized=false 时把用户送到 /install，
 * 让全新部署的运维直接落在初始化向导，而不是一个必然失败的登录表单。
 *
 * 探测失败（install 端点未部署、网络不可用）时按"已初始化"放行：守卫只是引导优化，
 * 不能因为它自身失败而把既有登录入口锁死（保守取向，见最终报告"歧义处置"）。
 */
export function InstallGuard({ children }: { children: ReactNode }) {
  const [state, setState] = useState<'loading' | 'allow' | 'redirect'>('loading')

  useEffect(() => {
    let alive = true
    getInstallStatus()
      .then((status) => {
        if (alive) setState(status.initialized ? 'allow' : 'redirect')
      })
      .catch(() => {
        if (alive) setState('allow')
      })
    return () => {
      alive = false
    }
  }, [])

  if (state === 'loading') {
    return (
      <div className="login-page">
        <div className="login-main">
          <div className="login-card">正在检查系统初始化状态…</div>
        </div>
      </div>
    )
  }
  if (state === 'redirect') {
    return <Navigate to="/install" replace />
  }
  return <>{children}</>
}
