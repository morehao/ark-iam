import { afterEach, expect, it } from 'vitest'
import { getCurrentTenantId, oidcExtraQueryParams, setCurrentTenantId } from './tenant'

afterEach(() => setCurrentTenantId(undefined))

it('keeps the OIDC tenant parameter in sync when switching and clearing tenants', () => {
  setCurrentTenantId('tenant-one')
  expect(getCurrentTenantId()).toBe('tenant-one')
  expect(oidcExtraQueryParams.tenant).toBe('tenant-one')

  setCurrentTenantId('tenant-two')
  expect(oidcExtraQueryParams.tenant).toBe('tenant-two')

  setCurrentTenantId(undefined)
  expect(getCurrentTenantId()).toBe('')
  expect(oidcExtraQueryParams).not.toHaveProperty('tenant')
})
