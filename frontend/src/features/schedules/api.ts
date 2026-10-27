import { z } from 'zod'
import { list } from '../../lib/admin'
import type { ListParams } from '../../lib/admin'

export const scheduleStates = [
  'queued',
  'claimed',
  'dispatching',
  'done',
  'failed',
  'unknown',
] as const

export const scheduleSchema = z.object({
  id: z.number(),
  attemptId: z.number(),
  companyId: z.number().nullable(),
  companyName: z.string(),
  credentialId: z.number(),
  username: z.string(),
  destination: z.string(),
  correlationId: z.string(),
  scheduledAt: z.string(),
  dispatchedAt: z.string().nullable(),
  state: z.enum(scheduleStates),
  submissionState: z.enum(['pending', 'accepted', 'rejected', 'unknown']),
  createdAt: z.string(),
})

export type Schedule = z.infer<typeof scheduleSchema>
export type ScheduleListParams = ListParams & {
  credentialId?: number
  dueFrom?: string
  dueTo?: string
}

export const listSchedules = (
  params: ScheduleListParams,
  signal?: AbortSignal,
) => list('schedules', scheduleSchema, params, signal)
