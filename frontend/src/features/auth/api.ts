import { z } from 'zod'
import { isAxiosError } from 'axios'
import { api } from '../../lib/api'
export const roles = ['admin', 'operator', 'viewer'] as const
export const userSchema = z.object({
  id: z.number(),
  name: z.string(),
  email: z.string(),
  role: z.enum(roles),
  active: z.boolean(),
  createdAt: z.string(),
})
export type SystemUser = z.infer<typeof userSchema>
export async function getSession(signal?: AbortSignal) {
  try {
    return userSchema.parse((await api.get('/admin/auth/me', { signal })).data)
  } catch (error) {
    if (isAxiosError(error) && error.response?.status === 401) return null
    throw error
  }
}
export async function login(body: {
  email: string
  password: string
  factor?: string
  remember?: boolean
}) {
  return userSchema.parse((await api.post('/admin/auth/login', body)).data)
}
export async function mfaStatus() {
  return z
    .object({ enabled: z.boolean() })
    .parse((await api.get('/admin/auth/mfa')).data)
}
export async function beginMFA(currentPassword: string) {
  return z
    .object({ secret: z.string(), uri: z.string() })
    .parse((await api.post('/admin/auth/mfa/setup', { currentPassword })).data)
}
export async function confirmMFA(code: string) {
  return z
    .object({ recoveryCodes: z.array(z.string()) })
    .parse((await api.post('/admin/auth/mfa/confirm', { code })).data)
}
export async function disableMFA(currentPassword: string, code: string) {
  await api.post('/admin/auth/mfa/disable', { currentPassword, code })
}
export async function requestPasswordRecovery(email: string) {
  await api.post('/admin/auth/recovery', { email })
}
export async function resetPassword(body: {
  token: string
  newPassword: string
  factor?: string
}) {
  await api.post('/admin/auth/recovery/reset', body)
}
export async function logout() {
  await api.post('/admin/auth/logout')
}
export async function changeOwnPassword(body: {
  currentPassword: string
  newPassword: string
}) {
  await api.post('/admin/auth/password', body)
}
