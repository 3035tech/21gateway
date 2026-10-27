import { z } from 'zod'
import { api } from '../../lib/api'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'
import { userSchema, roles } from '../auth/api'
export type { SystemUser } from '../auth/api'
export const systemPasswordLimits = {
  minimumCharacters: 15,
  maximumBytes: 128,
} as const

export const userFormSchema = z.object({
  name: z.string().trim().min(1, 'Informe o nome.').max(120),
  email: z.email('Informe um e-mail válido.').max(254),
  role: z.enum(roles),
  active: z.boolean(),
})
export type UserBody = z.infer<typeof userFormSchema>
export const listUsers = (params: ListParams, signal?: AbortSignal) =>
  list('users', userSchema, params, signal)
export async function saveUser(id: number | undefined, body: UserBody) {
  if (id) await api.put(`/admin/users/${id}`, body)
  else await api.post('/admin/users/invite', body)
}
export async function resendInvitation(id: number) {
  await api.post(`/admin/users/${id}/invite`)
}
export async function archiveUser(id: number) {
  await api.delete(`/admin/users/${id}`)
}
