import { PanelShell } from '../components/admin/PanelShell'
import type { PropsWithChildren } from 'react'
import { Navigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useUser } from '../features/auth/context'
import { logout } from '../features/auth/api'
import { errorMessage, roleLabels } from '../lib/admin'
export function AdminLayout({ children }: PropsWithChildren) {
  const user = useUser()
  const client = useQueryClient()
  const mutation = useMutation({
    mutationFn: logout,
    onSuccess: () => {
      client.removeQueries({ queryKey: ['admin'] })
      client.setQueryData(['session'], null)
    },
  })
  if (!user) return <Navigate to="/login" replace />
  return (
    <PanelShell
      name={user.name}
      role={roleLabels[user.role]}
      admin={user.role === 'admin'}
      pending={mutation.isPending}
      logoutError={mutation.isError ? errorMessage(mutation.error) : undefined}
      onLogout={() => mutation.mutate()}
    >
      {children}
    </PanelShell>
  )
}
