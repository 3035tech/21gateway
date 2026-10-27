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
  companyId: z.number().nullable(),
  companyName: z.string(),
  username: z.string(),
  active: z.boolean(),
  callbackUrl: z.string(),
  callbacksEnabled: z.boolean(),
  createdAt: z.string(),
})
export type Credential = z.infer<typeof credentialSchema>
export const credentialFormSchema = z
  .object({
    companyId: z.number().int().positive('Selecione uma empresa.'),
    username: z
      .string()
      .min(1)
      .max(120)
      .regex(/^[^:\r\n]+$/, 'Usuário inválido.'),
    preserveLegacyPassword: z.boolean(),
    legacyPassword: z.string().max(128),
    providerToken: z
      .string()
      .max(4096)
      .refine((value) => !/[\r\n]/.test(value), 'Token inválido.'),
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
      }, 'Use uma URL HTTPS válida.'),
    moCallbacksEnabled: z.boolean(),
    legacyAccountId: z
      .string()
      .max(64)
      .refine(
        (value) => !value || /^[0-9]+(_[0-9]+)?$/.test(value),
        'Use o ID legado, como 2340 ou 2340_779.',
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
      }, 'Use uma URL HTTPS válida.'),
    callbacksEnabled: z.boolean(),
  })
  .refine((value) => !value.callbacksEnabled || !!value.callbackUrl, {
    path: ['callbackUrl'],
    message: 'Informe o destino do callback.',
  })
  .refine((value) => !value.whitelistEnabled || !!value.allowedIPsText.trim(), {
    path: ['allowedIPsText'],
    message: 'Informe ao menos um IP ou rede.',
  })
  .refine((value) => !value.moCallbacksEnabled || !!value.moCallbackUrl, {
    path: ['moCallbackUrl'],
    message: 'Informe a URL de respostas MO.',
  })
  .refine((value) => !value.clearProviderToken || !value.providerToken, {
    path: ['providerToken'],
    message: 'Escolha substituir ou remover o token.',
  })
  .refine((value) => !value.preserveLegacyPassword || !!value.legacyPassword, {
    path: ['legacyPassword'],
    message: 'Informe a senha usada atualmente na API 21.',
  })
export type CredentialBody = z.infer<typeof credentialFormSchema>
const secretSchema = z.object({
  credential: credentialSchema,
  password: z.string(),
})
export const listCredentials = (params: ListParams, signal?: AbortSignal) =>
  list('credentials', credentialSchema, params, signal)
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
    ...fields
  } = body
  const payload = {
    ...fields,
    legacyPassword: preserveLegacyPassword ? legacyPassword : undefined,
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
