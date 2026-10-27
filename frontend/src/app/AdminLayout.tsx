import { PanelShell } from '../components/admin/PanelShell'
import type { PropsWithChildren } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useUser } from '../features/auth/context'
import { logout } from '../features/auth/api'
import { errorMessage, roleLabels } from '../lib/admin'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { listCompanies } from '../features/companies/api'
import { CompanyViewContext } from '../features/companies/CompanyViewContext'
export function AdminLayout({ children }: PropsWithChildren) {
  const { t } = useTranslation()
  const user = useUser()
  const client = useQueryClient()
  const navigate = useNavigate()
  const location = useLocation()
  const [selection, setSelection] = useState<{
    userId: number
    companyId: number
  }>()
  const companyIds = user?.role === 'admin' ? [] : (user?.companyIds ?? [])
  const selectedCompanyId =
    selection?.userId === user?.id &&
    companyIds.includes(selection?.companyId ?? 0)
      ? selection?.companyId
      : companyIds[0]
  const companies = useQuery({
    queryKey: ['admin', 'view-company-options', user?.id, companyIds],
    enabled: companyIds.length > 1,
    staleTime: 60_000,
    queryFn: async ({ signal }) => {
      const names = new Map<number, string>()
      for (let page = 1; ; page++) {
        const result = await listCompanies(
          { page, pageSize: 100, search: '', sort: 'name', direction: 'asc' },
          signal,
        )
        result.items.forEach((company) => names.set(company.id, company.name))
        if (page * 100 >= result.total || result.items.length === 0)
          return names
      }
    },
  })
  const mutation = useMutation({
    mutationFn: logout,
    onSuccess: () => {
      client.removeQueries({ queryKey: ['admin'] })
      client.setQueryData(['session'], null)
    },
  })
  if (!user) return <Navigate to="/login" replace />
  return (
    <CompanyViewContext.Provider value={selectedCompanyId}>
      <PanelShell
        name={user.name}
        role={t(roleLabels[user.role])}
        admin={user.role === 'admin'}
        pending={mutation.isPending}
        logoutError={
          mutation.isError ? errorMessage(mutation.error) : undefined
        }
        onLogout={() => mutation.mutate()}
        companyView={
          companyIds.length > 1
            ? {
                value: selectedCompanyId ?? companyIds[0],
                options: companyIds.map((id) => ({
                  value: id,
                  label:
                    companies.data?.get(id) ??
                    t('common.companyNumber', { id }),
                })),
                onChange: (companyId) => {
                  setSelection({
                    userId: user.id,
                    companyId: Number(companyId),
                  })
                  if (/^\/empresas\/\d+$/.test(location.pathname))
                    navigate(`/empresas/${companyId}`)
                  if (/^\/credenciais\/\d+$/.test(location.pathname))
                    navigate('/credenciais')
                },
              }
            : undefined
        }
      >
        {children}
      </PanelShell>
    </CompanyViewContext.Provider>
  )
}
