import { z } from 'zod'
import { api } from '../../lib/api'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'
export const credentialSchema = z.object({
  providerTokenConfigured: z.boolean(),
  whitelistEnabled: z.boolean(),
  allowedIPs: z.array(z.string()),
  moCallbackUrl: z.string(),
  moCallbacksEnabled: z.boolean(),
  legacyAccountId: z.string(),
  id: z.number(),
  publicRef: z.uuid(),
  companyId: z.number().nullable(),
  companyName: z.string(),
  username: z.string(),
  active: z.boolean(),
  callbackUrl: z.string(),
  callbacksEnabled: z.boolean(),
  createdAt: z.string(),
  expiresAt: z.string().nullable(),
})
export type Credential = z.infer<typeof credentialSchema>
export const credentialFormSchema = z
  .object({
    companyId: z.number().int().positive('validation.selectCompany'),
    generateUsername: z.boolean(),
    passwordMode: z.enum(['generated', 'manual', 'legacy']),
    username: z
      .string()
      .max(120)
      .regex(/^[^:\r\n]*$/, 'validation.invalidUsername'),
    preserveLegacyPassword: z.boolean(),
    legacyPassword: z.string().max(128),
    manualPassword: z.string().max(128),
    expiresAtText: z
      .string()
      .refine(
        (value) =>
          !value ||
          (!Number.isNaN(new Date(value).getTime()) &&
            new Date(value).getTime() > Date.now()),
        'validation.futureExpiry',
      ),
    providerToken: z
      .string()
      .max(4096)
      .refine((value) => !/[\r\n]/.test(value), 'validation.invalidToken'),
    clearProviderToken: z.boolean(),
    whitelistEnabled: z.boolean(),
    allowedIPsText: z.string().max(8192),
    moCallbackUrl: z
      .string()
      .max(2048)
      .refine((value) => {
        if (!value) return true
        try {
          const u = new URL(value)
          return (
            u.protocol === 'https:' &&
            !u.username &&
            !u.password &&
            !u.hash &&
            (!u.port || u.port === '443')
          )
        } catch {
          return false
        }
      }, 'validation.httpsUrl'),
    moCallbacksEnabled: z.boolean(),
    legacyAccountId: z
      .string()
      .max(64)
      .refine(
        (value) => !value || /^[0-9]+(_[0-9]+)?$/.test(value),
        'validation.legacyId',
      ),
    active: z.boolean(),
    callbackUrl: z
      .string()
      .max(2048)
      .refine((value) => {
        if (!value) return true
        try {
          const u = new URL(value)
          return (
            u.protocol === 'https:' &&
            !u.username &&
            !u.password &&
            !u.hash &&
            (!u.port || u.port === '443')
          )
        } catch {
          return false
        }
      }, 'validation.httpsUrl'),
    callbacksEnabled: z.boolean(),
  })
  .refine((value) => !value.callbacksEnabled || !!value.callbackUrl, {
    path: ['callbackUrl'],
    message: 'validation.callbackDestination',
  })
  .refine((value) => !value.whitelistEnabled || !!value.allowedIPsText.trim(), {
    path: ['allowedIPsText'],
    message: 'validation.ipRequired',
  })
  .refine((value) => !value.moCallbacksEnabled || !!value.moCallbackUrl, {
    path: ['moCallbackUrl'],
    message: 'validation.moUrl',
  })
  .refine((value) => !value.clearProviderToken || !value.providerToken, {
    path: ['providerToken'],
    message: 'validation.tokenChoice',
  })
  .refine((value) => !value.preserveLegacyPassword || !!value.legacyPassword, {
    path: ['legacyPassword'],
    message: 'validation.legacyPassword',
  })
  .refine((value) => value.generateUsername || !!value.username.trim(), {
    path: ['username'],
    message: 'validation.invalidUsername',
  })
  .refine(
    (value) =>
      !value.manualPassword ||
      Array.from(value.manualPassword.trim()).length >= 15,
    {
      path: ['manualPassword'],
      message: 'validation.passwordMinimumTrimmed',
    },
  )
  .refine((value) => !value.manualPassword || !value.preserveLegacyPassword, {
    path: ['manualPassword'],
    message: 'validation.passwordModeConflict',
  })
  .refine(
    (value) => value.passwordMode !== 'manual' || !!value.manualPassword,
    {
      path: ['manualPassword'],
      message: 'validation.manualPasswordRequired',
    },
  )
export type CredentialBody = z.infer<typeof credentialFormSchema>
const secretSchema = z.object({
  credential: credentialSchema,
  password: z.string(),
})
export const listCredentials = (params: ListParams, signal?: AbortSignal) =>
  list('credentials', credentialSchema, params, signal)
export async function getCredential(id: number, signal?: AbortSignal) {
  return credentialSchema.parse(
    (await api.get(`/admin/credentials/${id}`, { signal })).data,
  )
}
export async function getCredentialByPublicRef(
  publicRef: string,
  signal?: AbortSignal,
) {
  return credentialSchema.parse(
    (await api.get(`/admin/credentials/by-ref/${publicRef}`, { signal })).data,
  )
}
export async function saveCredential(
  id: number | undefined,
  body: CredentialBody,
) {
  const {
    allowedIPsText,
    providerToken,
    legacyAccountId,
    preserveLegacyPassword,
    legacyPassword,
    manualPassword,
    expiresAtText,
    generateUsername,
    passwordMode,
    ...fields
  } = body
  const payload = {
    ...fields,
    legacyPassword:
      preserveLegacyPassword && passwordMode === 'legacy'
        ? legacyPassword
        : undefined,
    manualPassword: passwordMode === 'manual' ? manualPassword : undefined,
    generateUsername: !id && generateUsername,
    username: generateUsername && !id ? '' : body.username,
    expiresAt: expiresAtText
      ? new Date(expiresAtText).toISOString()
      : undefined,
    clearExpiry: !!id && !expiresAtText,
    allowedIPs: allowedIPsText.split(/[\s,;]+/).filter(Boolean),
    providerToken: providerToken || undefined,
    legacyAccountId: legacyAccountId || undefined,
  }
  return secretSchema.parse(
    (id
      ? await api.put(`/admin/credentials/${id}`, payload)
      : await api.post('/admin/credentials', payload)
    ).data,
  )
}
export async function rotateCredential(id: number) {
  return secretSchema.parse(
    (await api.post(`/admin/credentials/${id}/rotate`)).data,
  )
}
export async function archiveCredential(id: number) {
  await api.delete(`/admin/credentials/${id}`)
}
