import axios from 'axios'
import { z } from 'zod'
import i18n from './i18n'

const baseURL = z
  .string()
  .url()
  .parse(import.meta.env.VITE_API_URL ?? 'http://localhost:8080')
export const api = axios.create({
  baseURL,
  timeout: 10_000,
  withCredentials: true,
})
api.interceptors.request.use((config) => {
  config.headers.set('Accept-Language', i18n.language)
  return config
})

type RequestConfig = { signal?: AbortSignal; timeout?: number; params?: Record<string, unknown> }

export async function apiGet<T>(
  url: string,
  schema: z.ZodType<T>,
  config?: RequestConfig,
): Promise<T> {
  const { data } = await api.get<unknown>(url, config)
  return schema.parse(data)
}

export async function apiPost<T>(
  url: string,
  schema: z.ZodType<T>,
  body?: unknown,
  config?: RequestConfig,
): Promise<T> {
  const { data } = await api.post<unknown>(url, body, config)
  return schema.parse(data)
}

const healthSchema = z.object({ status: z.literal('ok') })
export async function getReadiness(signal?: AbortSignal) {
  return apiGet('/readyz', healthSchema, { signal })
}
