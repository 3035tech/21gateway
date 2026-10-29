import { z } from 'zod'
import { api } from '../../lib/api'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'
export const credentialSchema = z.object({
  callbackAuth: z
    .object({
      mode: z.enum(['none', 'header', 'basic']),
      headerName: z.string(),
      username: z.string(),
      secretConfigured: z.boolean(),
    })
    .optional(),
  moCallbackAuth: z
    .object({
      mode: z.enum(['none', 'header', 'basic']),
      headerName: z.string(),
      username: z.string(),
      secretConfigured: z.boolean(),
    })
    .optional(),
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
const hasControlCharacter = (value: string, allowTab = false) =>
  Array.from(value).some((char) => {
    const code = char.charCodeAt(0)
    return (code < 32 && !(allowTab && code === 9)) || code === 127
  })
const callbackAuthFormSchema = z
  .object({
    mode: z.enum(['none', 'header', 'basic']),
    headerName: z.string().max(100),
    username: z.string().max(256),
    secret: z.string().max(4096),
    secretConfigured: z.boolean(),
    existingMode: z.enum(['none', 'header', 'basic']),
  })
  .superRefine((value, context) => {
    if (
      value.mode === 'header' &&
      (!/^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/.test(value.headerName) ||
        [
          'host',
          'content-length',
          'content-type',
          'connection',
          'transfer-encoding',
          'x-21mobile-event-id',
          'proxy-authorization',
        ].includes(value.headerName.toLowerCase()))
    ) {
      context.addIssue({
        code: 'custom',
        path: ['headerName'],
        message: 'validation.invalidHeaderName',
      })
    }
    if (
      value.mode === 'basic' &&
      (!value.username.trim() ||
        value.username.includes(':') ||
        hasControlCharacter(value.username))
    ) {
      context.addIssue({
        code: 'custom',
        path: ['username'],
        message: 'validation.invalidUsername',
      })
    }
    if (value.mode !== 'none' && hasControlCharacter(value.secret, true)) {
      context.addIssue({
        code: 'custom',
        path: ['secret'],
        message: 'validation.invalidCallbackSecret',
      })
    }
    if (
      value.mode !== 'none' &&
      !value.secret &&
      !(value.secretConfigured && value.mode === value.existingMode)
    ) {
      context.addIssue({
        code: 'custom',
        path: ['secret'],
        message: 'validation.required',
      })
    }
  })
export const credentialFormSchema = z
  .object({
    callbackAuth: callbackAuthFormSchema,
    moCallbackAuth: callbackAuthFormSchema,
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
    callbackAuth,
    moCallbackAuth,
    ...fields
  } = body
  const authPayload = (value: typeof callbackAuth) => ({
    mode: value.mode,
    headerName: value.mode === 'header' ? value.headerName : '',
    username: value.mode === 'basic' ? value.username : '',
    secret: value.secret || undefined,
  })
  const payload = {
    ...fields,
    callbackAuth: authPayload(callbackAuth),
    moCallbackAuth: authPayload(moCallbackAuth),
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
