import { expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { credentialFormSchema, saveCredential } from './api'
it('preserves existing secrets when empty and converts the whitelist to an array', async () => {
  const dto = {
    id: 1,
    companyId: 1,
    companyName: 'Example',
    username: 'client',
    active: true,
    callbackUrl: '',
    callbacksEnabled: false,
    createdAt: '2026-10-14T10:00:00Z',
    providerTokenConfigured: true,
    whitelistEnabled: true,
    allowedIPs: ['203.0.113.1/32'],
    moCallbackUrl: '',
    moCallbacksEnabled: false,
    legacyAccountId: '1',
  }
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { credential: dto, password: '' } })
  const body = credentialFormSchema.parse({
    companyId: 1,
    username: 'client',
    active: true,
    callbackUrl: '',
    callbacksEnabled: false,
    providerToken: '',
    clearProviderToken: false,
    whitelistEnabled: true,
    allowedIPsText: '203.0.113.1\n198.51.100.0/24',
    moCallbackUrl: '',
    moCallbacksEnabled: false,
    legacyAccountId: '1',
    preserveLegacyPassword: false,
    legacyPassword: '',
  })
  await saveCredential(1, body)
  expect(put.mock.calls[0][1]).toMatchObject({
    providerToken: undefined,
    legacyPassword: undefined,
    allowedIPs: ['203.0.113.1', '198.51.100.0/24'],
  })
  expect(put.mock.calls[0][1]).not.toHaveProperty('allowedIPsText')
  put.mockRestore()
})
it('requires an existing password when choosing to preserve legacy access', () => {
  const value = {
    companyId: 1,
    username: 'client',
    active: true,
    callbackUrl: '',
    callbacksEnabled: false,
    providerToken: '',
    clearProviderToken: false,
    whitelistEnabled: false,
    allowedIPsText: '',
    moCallbackUrl: '',
    moCallbacksEnabled: false,
    legacyAccountId: '',
    preserveLegacyPassword: true,
    legacyPassword: '',
  }
  expect(credentialFormSchema.safeParse(value).success).toBe(false)
  expect(
    credentialFormSchema.safeParse({ ...value, legacyPassword: 'old' }).success,
  ).toBe(true)
})
