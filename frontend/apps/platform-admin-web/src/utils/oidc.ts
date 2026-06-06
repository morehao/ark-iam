const OIDC_ISSUER = import.meta.env.VITE_OIDC_ISSUER ?? '/v1/iam/oidc'
const OIDC_CLIENT_ID = import.meta.env.VITE_OIDC_CLIENT_ID ?? 'platform-admin-web'
const OIDC_REDIRECT_URI = `${window.location.origin}/auth/callback`
const OIDC_SCOPE = 'openid profile email'

function base64URLEncode(buffer: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(buffer)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

async function sha256(plain: string): Promise<ArrayBuffer> {
  const encoder = new TextEncoder()
  return crypto.subtle.digest('SHA-256', encoder.encode(plain))
}

export interface PKCEParams {
  codeVerifier: string
  codeChallenge: string
  state: string
}

export function generatePKCEParams(): PKCEParams {
  const array = new Uint8Array(32)
  crypto.getRandomValues(array)
  const codeVerifier = base64URLEncode(array.buffer)
  const state = crypto.randomUUID()

  return { codeVerifier, codeChallenge: '', state }
}

export async function generateCodeChallenge(verifier: string): Promise<string> {
  return base64URLEncode(await sha256(verifier))
}

export function buildAuthorizeURL(params: PKCEParams): string {
  const url = new URL(`${OIDC_ISSUER}/authorize`, window.location.origin)
  url.searchParams.set('client_id', OIDC_CLIENT_ID)
  url.searchParams.set('redirect_uri', OIDC_REDIRECT_URI)
  url.searchParams.set('response_type', 'code')
  url.searchParams.set('scope', OIDC_SCOPE)
  url.searchParams.set('state', params.state)
  url.searchParams.set('code_challenge', params.codeChallenge)
  url.searchParams.set('code_challenge_method', 'S256')
  return url.toString()
}

export function storePKCEParams(params: PKCEParams): void {
  sessionStorage.setItem('oidc_pkce', JSON.stringify(params))
}

export function loadPKCEParams(): PKCEParams | null {
  const raw = sessionStorage.getItem('oidc_pkce')
  if (!raw) return null
  try {
    return JSON.parse(raw) as PKCEParams
  } catch {
    return null
  }
}

export function clearPKCEParams(): void {
  sessionStorage.removeItem('oidc_pkce')
}

export interface TokenResponse {
  access_token: string
  id_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export async function exchangeCodeForTokens(code: string, codeVerifier: string): Promise<TokenResponse> {
  const form = new URLSearchParams()
  form.append('grant_type', 'authorization_code')
  form.append('code', code)
  form.append('redirect_uri', OIDC_REDIRECT_URI)
  form.append('client_id', OIDC_CLIENT_ID)
  form.append('code_verifier', codeVerifier)

  const resp = await fetch(`${OIDC_ISSUER}/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form.toString(),
  })

  if (!resp.ok) {
    throw new Error(`Token exchange failed: ${resp.status}`)
  }
  return resp.json()
}

export async function refreshTokens(refreshToken: string): Promise<TokenResponse> {
  const form = new URLSearchParams()
  form.append('grant_type', 'refresh_token')
  form.append('refresh_token', refreshToken)
  form.append('client_id', OIDC_CLIENT_ID)

  const resp = await fetch(`${OIDC_ISSUER}/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form.toString(),
  })

  if (!resp.ok) {
    throw new Error(`Token refresh failed: ${resp.status}`)
  }
  return resp.json()
}

export function parseJWT(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = parts[1]
    return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
  } catch {
    return null
  }
}

export function getEndSessionURL(idToken: string): string {
  const url = new URL(`${OIDC_ISSUER}/end_session`, window.location.origin)
  url.searchParams.set('id_token_hint', idToken)
  url.searchParams.set('post_logout_redirect_uri', `${window.location.origin}/login`)
  return url.toString()
}
