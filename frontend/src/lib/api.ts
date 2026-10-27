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
const healthSchema = z.object({ status: z.literal('ok') })
export async function getReadiness(signal?: AbortSignal) {
  const { data } = await api.get<unknown>('/readyz', { signal })
  return healthSchema.parse(data)
}
