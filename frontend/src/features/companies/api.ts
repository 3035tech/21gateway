import { z } from 'zod'
import { api } from '../../lib/api'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'
export const companySchema = z.object({
  providerTokenConfigured: z.boolean(),
  legacyProviderTokenFallback: z.boolean(),
  id: z.number(),
  publicRef: z.uuid(),
  name: z.string(),
  reference: z.string(),
  active: z.boolean(),
  createdAt: z.string(),
})
export type Company = z.infer<typeof companySchema>
export const companyFormSchema = z.object({
  providerToken: z
    .string()
    .max(4096)
    .refine((value) => !/[\r\n]/.test(value), 'validation.invalidToken'),
  clearProviderToken: z.boolean(),
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
export async function getCompanyByPublicRef(
  publicRef: string,
  signal?: AbortSignal,
) {
  const response = await api.get(`/admin/companies/by-ref/${publicRef}`, {
    signal,
  })
  return companySchema.parse(response.data)
}
export async function saveCompany(id: number | undefined, body: CompanyBody) {
  const payload = {
    name: body.name,
    reference: body.reference,
    active: body.active,
    ...(body.providerToken ? { providerToken: body.providerToken } : {}),
    ...(body.clearProviderToken ? { clearProviderToken: true } : {}),
  }
  if (id) await api.put(`/admin/companies/${id}`, payload)
  else await api.post('/admin/companies', payload)
}
export async function archiveCompany(id: number) {
  await api.delete(`/admin/companies/${id}`)
}
