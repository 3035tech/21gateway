import { z } from 'zod'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'
export const auditSchema = z.object({
  id: z.number(),
  actorId: z.number().nullable(),
  actorName: z.string().optional(),
  action: z.string(),
  resource: z.string(),
  resourceId: z.number().nullable(),
  createdAt: z.string(),
})
export type Audit = z.infer<typeof auditSchema>
export type AuditListParams = ListParams & {
  actorId?: number
  action?: string
  resource?: string
}
export const listAudit = (params: AuditListParams, signal?: AbortSignal) =>
  list('audit', auditSchema, params, signal)
