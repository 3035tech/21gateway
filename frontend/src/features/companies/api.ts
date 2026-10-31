import { z } from 'zod'
import { api } from '../../lib/api'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'
import { availableSMSProviders, SMS_PROVIDER_ZOUG } from './providers'
export const companySchema = z.object({
  providerTokenConfigured: z.boolean(),
  legacyProviderTokenFallback: z.boolean(),
  smsProvider: z.string().min(1).default(SMS_PROVIDER_ZOUG),
  id: z.number(),
  publicRef: z.uuid(),
  name: z.string(),
  reference: z.string(),
  active: z.boolean(),
  createdAt: z.string(),
})
export type Company = z.infer<typeof companySchema>
export const companyFormSchema = z.object({
  smsProvider: z.enum(availableSMSProviders).optional(),
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
const companyHealthSchema = z.object({
  activeCredentials: z.number(),
  dlrConfigured: z.number(),
  moConfigured: z.number(),
  expiringSoon: z.number(),
  unknownSends: z.number(),
  pendingCallbacks: z.number(),
  failedCallbacks: z.number(),
  lastSendAt: z.string().nullable(),
  lastProviderAt: z.string().nullable(),
  lastClientDlrAt: z.string().nullable(),
  lastClientMoAt: z.string().nullable(),
})
export type CompanyHealth = z.infer<typeof companyHealthSchema>
export async function getCompanyHealth(
  publicRef: string,
  signal?: AbortSignal,
) {
  const response = await api.get(
    `/admin/companies/by-ref/${publicRef}/health`,
    { signal },
  )
  return companyHealthSchema.parse(response.data)
}
export async function saveCompany(id: number | undefined, body: CompanyBody) {
  const payload = {
    name: body.name,
    reference: body.reference,
    active: body.active,
    smsProvider: body.smsProvider ?? SMS_PROVIDER_ZOUG,
    ...(body.providerToken ? { providerToken: body.providerToken } : {}),
    ...(body.clearProviderToken ? { clearProviderToken: true } : {}),
  }
  if (id) await api.put(`/admin/companies/${id}`, payload)
  else await api.post('/admin/companies', payload)
}
export async function archiveCompany(id: number) {
  await api.delete(`/admin/companies/${id}`)
}

const companyTestSendSchema = z.object({
  destination: z.string(),
  correlationId: z.string(),
  providerMessageId: z.string().nullable(),
  state: z.enum(['accepted', 'rejected', 'unknown']),
  failureCode: z.string(),
  createdAt: z.string(),
})
export async function getCompanyTestSends(
  publicRef: string,
  signal?: AbortSignal,
) {
  const response = await api.get(
    `/admin/companies/by-ref/${publicRef}/test-sends`,
    { signal },
  )
  return z.array(companyTestSendSchema).parse(response.data)
}

const zougDiagnosticSchema = z.object({
  runId: z.string(),
  results: z.array(
    z.object({
      caseId: z.string(),
      requestMethod: z.string(),
      requestPath: z.string(),
      requestBody: z.string(),
      contentType: z.string(),
      auth: z.string(),
      httpStatus: z.number(),
      responseBody: z.string(),
      responseHeaders: z.record(z.string(), z.array(z.string())).optional(),
      durationMs: z.number(),
      errorCode: z.string().optional(),
    }),
  ),
})
export async function runZougDiagnostic(
  publicRef: string,
  destination: string,
) {
  const response = await api.post(
    `/admin/companies/by-ref/${publicRef}/zoug-diagnostic`,
    { destination },
    { timeout: 28_000 },
  )
  return zougDiagnosticSchema.parse(response.data)
}
