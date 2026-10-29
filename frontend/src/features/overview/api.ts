import { z } from 'zod'
import { api } from '../../lib/api'
const schema = z.object({
  companies: z.number(),
  activeCredentials: z.number(),
  accepted: z.number(),
  unknown: z.number(),
  rejected: z.number(),
  pendingCallbacks: z.number(),
  failedCallbacks: z.number(),
  pendingMoCallbacks: z.number(),
  failedMoCallbacks: z.number(),
  scheduledQueued: z.number(),
  scheduledDue: z.number(),
  scheduledNext24h: z.number(),
  scheduledActive: z.number(),
})
export async function getSummary(companyId?: number, signal?: AbortSignal) {
  return schema.parse(
    (await api.get('/admin/summary', { params: { companyId }, signal })).data,
  )
}

const queueSchema = z.object({
  unmatchedEvents: z.number(), oldestUnmatchedAt: z.string().nullable(),
  pending: z.number(), processing: z.number(), expiredLeases: z.number(),
  failedLastHour: z.number(), oldestPendingAt: z.string().nullable(),
})
const operationsSchema = z.object({
  observedAt: z.string(), dlr: queueSchema, mo: queueSchema,
  unresolvedSends: z.number(), oldestUnresolvedAt: z.string().nullable(),
  scheduler: z.object({
    ready: z.number(), openBatches: z.number(), unpublishedBatches: z.number(),
    oldestOpenBatchAt: z.string().nullable(), oldestUnpublishedAt: z.string().nullable(),
    queued: z.number(), due: z.number(), claimed: z.number(),
    dispatching: z.number(), unknownLastHour: z.number(), failedLastHour: z.number(),
    oldestDueAt: z.string().nullable(),
  }),
})
export type OperationsSnapshot = z.infer<typeof operationsSchema>
export async function getOperations(signal?: AbortSignal) {
  return operationsSchema.parse((await api.get('/admin/operations', { signal })).data)
}
