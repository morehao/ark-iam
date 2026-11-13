export const oidcExtraQueryParams: Record<string, string | number | boolean> = {}

let currentTenantId = ''

export function getCurrentTenantId(): string {
  return currentTenantId
}

export function setCurrentTenantId(id: string | number | undefined) {
  const next = id == null ? '' : String(id)
  currentTenantId = next
  if (next) {
    oidcExtraQueryParams.tenant = next
  } else {
    delete oidcExtraQueryParams.tenant
  }
}
