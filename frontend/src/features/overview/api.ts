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
export async function getSummary(signal?: AbortSignal) {
  return schema.parse((await api.get('/admin/summary', { signal })).data)
}
