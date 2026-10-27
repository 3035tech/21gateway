import { z } from 'zod'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'
export const messageSchema = z.object({
  id: z.number(),
  credentialId: z.number(),
  companyId: z.number().nullable(),
  companyName: z.string().optional(),
  destination: z.string(),
  correlationId: z.string(),
  state: z.enum(['pending', 'accepted', 'rejected', 'unknown']),
  failureCode: z.string(),
  deliveryStatus: z.string(),
  createdAt: z.string(),
})
export type Message = z.infer<typeof messageSchema>
export const listMessages = (params: ListParams, signal?: AbortSignal) =>
  list('messages', messageSchema, params, signal)
