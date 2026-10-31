import { z } from 'zod'
import { api } from '../../lib/api'

export const caseKinds = [
  'unknown_send',
  'overdue_schedule',
  'dlr_callback',
  'mo_callback',
] as const
export const caseStatuses = ['open', 'in_progress', 'completed'] as const
export type CaseKind = (typeof caseKinds)[number]
export type CaseStatus = (typeof caseStatuses)[number]

const itemSchema = z.object({
  kind: z.enum(caseKinds),
  sourceId: z.number(),
  attemptId: z.number(),
  companyId: z.number(),
  companyName: z.string(),
  createdAt: z.string(),
  status: z.enum(caseStatuses),
  ownerId: z.number().nullable(),
  ownerName: z.string(),
  note: z.string(),
  tracked: z.boolean(),
  sourceActive: z.boolean(),
})
export type OperationCase = z.infer<typeof itemSchema>

export type CaseParams = {
  page: number
  pageSize: number
  search: string
  companyId?: number
  kind?: CaseKind
  status?: CaseStatus
  sort?: string
  direction: 'asc' | 'desc'
}
export async function listOperationCases(
  params: CaseParams,
  signal?: AbortSignal,
) {
  const response = await api.get('/admin/operation-cases', { params, signal })
  return z
    .object({
      items: z.array(itemSchema),
      total: z.number(),
      page: z.number(),
      pageSize: z.number(),
    })
    .parse(response.data)
}
export async function updateOperationCase(
  item: OperationCase,
  status: CaseStatus,
  note: string,
) {
  await api.put(`/admin/operation-cases/${item.kind}/${item.sourceId}`, {
    status,
    note,
  })
}
export async function operationCaseHistory(
  item: OperationCase,
  signal?: AbortSignal,
) {
  const response = await api.get(
    `/admin/operation-cases/${item.kind}/${item.sourceId}/history`,
    { signal },
  )
  return z
    .array(
      z.object({
        actorName: z.string(),
        status: z.enum(caseStatuses),
        note: z.string(),
        createdAt: z.string(),
      }),
    )
    .parse(response.data)
}
