import { z } from 'zod'
import { api } from '../../lib/api'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'
export const callbackSchema = z.object({
  id: z.number(),
  attemptId: z.number(),
  credentialId: z.number(),
  companyId: z.number().nullable(),
  companyName: z.string().optional(),
  url: z.string(),
  status: z.string(),
  state: z.enum(['pending', 'processing', 'delivered', 'failed']),
  attempts: z.number(),
  lastError: z.string(),
  availableAt: z.string(),
  createdAt: z.string(),
})
export type Callback = z.infer<typeof callbackSchema>
export const listCallbacks = (params: ListParams, signal?: AbortSignal) =>
  list('callbacks', callbackSchema, params, signal)
export async function retryCallback(id: number) {
  await api.post(`/admin/callbacks/${id}/retry`)
}
