import { z } from 'zod'
import { api } from '../../lib/api'

const itemSchema = z.object({
  id: z.number(),
  type: z.string(),
  title: z.string(),
  description: z.string(),
  path: z.string(),
  openedAt: z.string(),
  resolvedAt: z.string().nullable(),
  readAt: z.string().nullable(),
})
const listSchema = z.object({
  items: z.array(itemSchema),
  unreadCount: z.number(),
  total: z.number(),
  page: z.number(),
  pageSize: z.number(),
})
export type NotificationItem = z.infer<typeof itemSchema>
export type NotificationList = z.infer<typeof listSchema>

export async function listNotifications(
  params: {
    page: number
    pageSize: number
    unread: boolean
    sort: 'openedAt' | 'title'
    order: 'asc' | 'desc'
  },
  signal?: AbortSignal,
) {
  return listSchema.parse(
    (await api.get('/admin/notifications', { params, signal })).data,
  )
}
export async function markNotificationRead(id: number) {
  await api.post(`/admin/notifications/${id}/read`)
}

const preferenceSchema = z.object({
  code: z.string(),
  title: z.string(),
  emailEnabled: z.boolean(),
})
export type NotificationPreference = z.infer<typeof preferenceSchema>
export async function listNotificationPreferences(signal?: AbortSignal) {
  return z
    .object({ items: z.array(preferenceSchema), emailAvailable: z.boolean() })
    .parse((await api.get('/admin/notification-preferences', { signal })).data)
}
export async function setNotificationPreference(
  code: string,
  enabled: boolean,
) {
  return preferenceSchema.parse(
    (
      await api.put(`/admin/notification-preferences/${code}`, {
        emailEnabled: enabled,
      })
    ).data,
  )
}
