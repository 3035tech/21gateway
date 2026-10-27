import { SessionState } from './SessionState'
import { useEffect } from 'react'
import type { PropsWithChildren } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../lib/api'
import { getSession } from './api'
import { AuthContext } from './context'
export function AuthProvider({ children }: PropsWithChildren) {
  const client = useQueryClient()
  const session = useQuery({
    queryKey: ['session'],
    queryFn: ({ signal }) => getSession(signal),
    retry: false,
    staleTime: 0,
    refetchOnWindowFocus: true,
  })
  useEffect(() => {
    const id = api.interceptors.response.use(
      (response) => response,
      (error) => {
        if (
          error.response?.status === 401 &&
          !error.config?.url?.includes('/auth/')
        ) {
          client.setQueryData(['session'], null)
          client.removeQueries({ queryKey: ['admin'] })
        }
        return Promise.reject(error)
      },
    )
    return () => api.interceptors.response.eject(id)
  }, [client])
  if (session.isPending || (session.isError && session.isFetching))
    return <SessionState checking />
  if (session.isError)
    return (
      <SessionState checking={false} onRetry={() => void session.refetch()} />
    )
  return (
    <AuthContext.Provider value={session.data ?? null}>
      {children}
    </AuthContext.Provider>
  )
}
