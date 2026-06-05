/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_SSO_CONNECTOR_ID?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
