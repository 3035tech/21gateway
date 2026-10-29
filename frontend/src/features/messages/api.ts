import { z } from 'zod'
import { list } from '../../lib/admin'
import { api } from '../../lib/api'
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

const messageContentSchema = z.object({
  state: z.enum(['available', 'expired', 'unavailable']),
  messageText: z.string().optional(),
})
export async function getMessageContent(id: number, signal?: AbortSignal) {
  const response = await api.get(`/admin/messages/${id}/content`, { signal })
  return messageContentSchema.parse(response.data)
}

const messageTraceSchema = z.object({
  id: z.number(),
  submissionState: messageSchema.shape.state,
  failureCode: z.string(),
  deliveryStatus: z.string(),
  providerMessageId: z.string().nullable().optional(),
  createdAt: z.string(),
  dispatchedAt: z.string().nullable().optional(),
  truncated: z.boolean(),
  events: z.array(z.object({
    kind: z.string(), id: z.number(), status: z.string(),
    occurredAt: z.string(), recordedAt: z.string(),
    attempts: z.number().optional(), failureCode: z.string().optional(),
  })),
})
export type MessageTrace = z.infer<typeof messageTraceSchema>
export async function getMessageTrace(id: number, signal?: AbortSignal) {
  const response = await api.get(`/admin/messages/${id}/trace`, { signal })
  return messageTraceSchema.parse(response.data)
}

const testSendResultSchema = z.object({
  submissionState: z.enum(['pending', 'accepted', 'rejected', 'unknown']),
  idSms: z.string().optional(),
  correlationId: z.string(),
  failureCode: z.string().optional(),
})
export type TestSendResult = z.infer<typeof testSendResultSchema>
export async function sendTestSMS(
  credentialId: number,
  destination: string,
  messageText?: string,
) {
  const response = await api.post<unknown>(
    `/admin/credentials/${credentialId}/test-send`,
    { destination, ...(messageText !== undefined ? { messageText } : {}) },
    { timeout: 20_000 },
  )
  return testSendResultSchema.parse(response.data)
}

export async function sendCompanyTokenTestSMS(
  publicRef: string,
  destination: string,
  messageText?: string,
) {
  const response = await api.post<unknown>(
    `/admin/companies/by-ref/${publicRef}/test-send`,
    { destination, ...(messageText !== undefined ? { messageText } : {}) },
    { timeout: 20_000 },
  )
  return testSendResultSchema.parse(response.data)
}
