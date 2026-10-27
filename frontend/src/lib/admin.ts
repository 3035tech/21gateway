import { isAxiosError } from 'axios'
import { z } from 'zod'
import { api } from './api'
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
const publicErrors: Record<string, string> = {
  UNAUTHORIZED: 'Sua sessão expirou. Entre novamente.',
  FORBIDDEN: 'Você não tem permissão para essa ação.',
  CONFLICT:
    'Já existe um cadastro com esses dados, ou a operação conflita com o estado atual.',
  LAST_ADMIN_REQUIRED: 'Mantenha pelo menos um administrador ativo.',
  INVALID_REQUEST: 'Confira os campos informados.',
  RATE_LIMITED: 'Muitas tentativas. Aguarde um minuto e tente novamente.',
}
export function errorMessage(error: unknown) {
  if (isAxiosError(error)) {
    const code = error.response?.data?.code
    if (typeof code === 'string' && publicErrors[code])
      return publicErrors[code]
  }
  return 'Não foi possível concluir. Tente novamente.'
}
export const formatDate = (value: string) =>
  new Date(value).toLocaleString('pt-BR')
export const roleLabels = {
  admin: 'Administrador',
  operator: 'Operador',
  viewer: 'Consulta',
}
