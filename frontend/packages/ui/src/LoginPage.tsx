import { Button, Typography } from 'antd'
import { ArrowRightOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import { useAuth } from 'react-oidc-context'
import { brand } from './theme'

interface Props {
  title: string
  subtitle?: string
}

export function LoginPage({ title, subtitle }: Props) {
  const auth = useAuth()

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'stretch',
        background: brand.gradientSoft,
      }}
    >
      {/* 左侧品牌区 */}
      <div
        style={{
          flex: 1.2,
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          padding: '0 8vw',
          background: brand.gradient,
          color: '#fff',
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            position: 'absolute',
            width: 420,
            height: 420,
            borderRadius: '50%',
            background: 'rgba(255,255,255,0.08)',
            top: -120,
            right: -120,
          }}
        />
        <div
          style={{
            position: 'absolute',
            width: 260,
            height: 260,
            borderRadius: '50%',
            background: 'rgba(255,255,255,0.06)',
            bottom: -60,
            left: -60,
          }}
        />
        <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 28, position: 'relative' }}>
          <div
            style={{
              width: 52,
              height: 52,
              borderRadius: 14,
              background: 'rgba(255,255,255,0.18)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 26,
              backdropFilter: 'blur(4px)',
            }}
          >
            <SafetyCertificateOutlined />
          </div>
          <div>
            <div style={{ fontSize: 22, fontWeight: 700 }}>{title}</div>
            {subtitle && <div style={{ fontSize: 13, opacity: 0.85 }}>{subtitle}</div>}
          </div>
        </div>
        <Typography.Title level={2} style={{ color: '#fff', marginBottom: 12, position: 'relative' }}>
          统一身份认证平台
        </Typography.Title>
        <Typography.Paragraph style={{ color: 'rgba(255,255,255,0.82)', fontSize: 15, maxWidth: 480, position: 'relative' }}>
          一个平台管理多应用的账号、权限与安全策略。基于标准 OIDC/OAuth2 协议，
          支持单点登录、多租户切换与细粒度权限控制。
        </Typography.Paragraph>
        <div style={{ display: 'flex', gap: 24, marginTop: 28, position: 'relative' }}>
          {['单点登录 SSO', '多租户隔离', '细粒度授权'].map((f) => (
            <div
              key={f}
              style={{
                padding: '8px 16px',
                borderRadius: 20,
                background: 'rgba(255,255,255,0.14)',
                fontSize: 13,
                backdropFilter: 'blur(4px)',
              }}
            >
              {f}
            </div>
          ))}
        </div>
      </div>

      {/* 右侧登录区 */}
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40 }}>
        <div
          style={{
            width: 380,
            background: '#fff',
            borderRadius: 16,
            boxShadow: '0 20px 60px rgba(15,23,42,0.12)',
            padding: '40px 36px',
          }}
        >
          <div style={{ fontSize: 20, fontWeight: 700, marginBottom: 8 }}>欢迎回来</div>
          <div style={{ fontSize: 14, color: brand.textSecondary, marginBottom: 32 }}>
            使用统一身份账号登录 {title}
          </div>
          <Button
            type="primary"
            size="large"
            block
            loading={auth.isLoading}
            onClick={() => void auth.signinRedirect()}
            style={{ height: 46, borderRadius: 10, fontWeight: 600, background: brand.gradient, border: 'none', boxShadow: '0 8px 20px rgba(79,110,247,0.35)' }}
          >
            IAM 账号登录 <ArrowRightOutlined />
          </Button>
          <div style={{ marginTop: 24, textAlign: 'center', fontSize: 12, color: brand.textSecondary }}>
            登录即代表同意平台的安全与隐私策略
          </div>
        </div>
      </div>
    </div>
  )
}
