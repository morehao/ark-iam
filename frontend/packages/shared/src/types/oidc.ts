export interface PKCEParams {
  codeVerifier: string
  codeChallenge: string
  state: string
}

export interface OIDCTokenResponse {
  access_token: string
  id_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export type OIDCFlowMode = 'interactive' | 'silent'
