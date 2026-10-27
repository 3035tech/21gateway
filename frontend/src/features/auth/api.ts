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
export async function login(body: { email: string; password: string }) {
  return userSchema.parse((await api.post('/admin/auth/login', body)).data)
}
export async function logout() {
  await api.post('/admin/auth/logout')
}
