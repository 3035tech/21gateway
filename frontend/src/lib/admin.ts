import { isAxiosError } from 'axios'
import { z } from 'zod'
import { api } from './api'
import i18n from './i18n'
export type ListParams = {
  page: number
  pageSize: number
  search: string
  sort?: string
  direction: 'asc' | 'desc'
  active?: boolean
  companyId?: number
  state?: string
  callbackState?: string
}
export type Page<T> = {
  items: T[]
  total: number
  page: number
  pageSize: number
}
export async function list<T extends z.ZodType>(
  path: string,
  schema: T,
  params: ListParams,
  signal?: AbortSignal,
): Promise<Page<z.infer<T>>> {
  const response = await api.get(`/admin/${path}`, { params, signal })
  return z
    .object({
      items: z.array(schema),
      total: z.number(),
      page: z.number(),
      pageSize: z.number(),
    })
    .parse(response.data)
}
export function errorMessage(error: unknown) {
  if (isAxiosError(error)) {
    const code = error.response?.data?.code
    if (typeof code === 'string' && i18n.exists(`errors.${code}`))
      return i18n.t(`errors.${code}`)
  }
  return i18n.t('errors.fallback')
}
export const formatDate = (value: string) =>
  new Date(value).toLocaleString(i18n.language)
export const roleLabels = {
  admin: 'roles.admin',
  operator: 'roles.operator',
  viewer: 'roles.viewer',
} as const
export const roleDescriptions = {
  admin: 'roles.adminDescription',
  operator: 'roles.operatorDescription',
  viewer: 'roles.viewerDescription',
} as const
