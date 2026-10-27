import { z } from 'zod'
import { api } from '../../lib/api'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'
export const companySchema = z.object({
  id: z.number(),
  name: z.string(),
  reference: z.string(),
  active: z.boolean(),
  createdAt: z.string(),
})
export type Company = z.infer<typeof companySchema>
export const companyFormSchema = z.object({
  name: z.string().trim().min(1, 'validation.name').max(160),
  reference: z.string().trim().max(64),
  active: z.boolean(),
})
export type CompanyBody = z.infer<typeof companyFormSchema>
export const listCompanies = (params: ListParams, signal?: AbortSignal) =>
  list('companies', companySchema, params, signal)
export async function getCompany(id: number, signal?: AbortSignal) {
  const response = await api.get(`/admin/companies/${id}`, { signal })
  return companySchema.parse(response.data)
}
export async function saveCompany(id: number | undefined, body: CompanyBody) {
  if (id) await api.put(`/admin/companies/${id}`, body)
  else await api.post('/admin/companies', body)
}
export async function archiveCompany(id: number) {
  await api.delete(`/admin/companies/${id}`)
}
