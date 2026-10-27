import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { DateCell } from '../../components/admin/DateCell'
import { DataTable } from '../../components/data-table/DataTable'
import { useListState } from '../../components/data-table/useListState'
import { errorMessage } from '../../lib/admin'
import { listAudit } from './api'
import type { Audit, AuditListParams } from './api'

const actions: Record<string, string> = {
  created: 'Cadastro criado',
  updated: 'Cadastro atualizado',
  archived: 'Cadastro arquivado',
  password_rotated: 'Senha da API alterada',
  login: 'Login',
  callback_retried: 'Callback recolocado na fila',
}
const resources: Record<string, string> = {
  companies: 'Empresas',
  users: 'Usuários',
  credentials: 'Credenciais',
  callbacks: 'Callbacks',
}
const columns: ColumnDef<Audit>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'actorName',
    header: 'Responsável',
    cell: ({ row }) =>
      row.original.actorId
        ? `${row.original.actorName || 'Usuário'} #${row.original.actorId}`
        : 'Sistema',
  },
  {
    accessorKey: 'action',
    header: 'Ação',
    cell: ({ row }) => actions[row.original.action] ?? row.original.action,
  },
  {
    accessorKey: 'resource',
    header: 'Recurso',
    cell: ({ row }) =>
      resources[row.original.resource] ?? row.original.resource,
  },
  { accessorKey: 'resourceId', header: 'ID do recurso' },
  {
    accessorKey: 'createdAt',
    header: 'Data',
    cell: ({ row }) => <DateCell value={row.original.createdAt} />,
  },
]

export function AuditPage() {
  const table = useListState([{ id: 'createdAt', desc: true }])
  const [actorId, setActorId] = useState('')
  const [action, setAction] = useState('')
  const [resource, setResource] = useState('')
  const sort = table.sorting[0]
  const params: AuditListParams = {
    page: table.pagination.pageIndex + 1,
    pageSize: table.pagination.pageSize,
    search: table.search,
    sort: sort?.id,
    direction: sort?.desc ? 'desc' : 'asc',
    actorId: actorId ? Number(actorId) : undefined,
    action: action || undefined,
    resource: resource || undefined,
  }
  const query = useQuery({
    queryKey: ['admin', 'audit', params],
    queryFn: ({ signal }) => listAudit(params, signal),
  })
  const resetPage = () =>
    table.onPaginationChange({ ...table.pagination, pageIndex: 0 })

  return (
    <>
      <header className="mb-7">
        <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
          Auditoria
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-600">
          Consulte alterações e acessos da equipe. O nome exibido é o atual; o
          ID identifica a conta mesmo após uma mudança de nome.
        </p>
      </header>
      <DataTable
        {...table}
        caption="Eventos de auditoria"
        searchPlaceholder="Responsável, ação ou recurso"
        columns={columns}
        data={query.data?.items ?? []}
        total={query.data?.total ?? 0}
        getRowId={(row) => String(row.id)}
        isLoading={query.isPending}
        error={query.isError ? errorMessage(query.error) : undefined}
        filters={
          <>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              ID do responsável
              <input
                className="form-control w-36"
                type="number"
                min={1}
                value={actorId}
                onChange={(event) => {
                  setActorId(event.target.value)
                  resetPage()
                }}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              Ação
              <select
                className="form-control"
                value={action}
                onChange={(event) => {
                  setAction(event.target.value)
                  resetPage()
                }}
              >
                <option value="">Todas</option>
                {Object.entries(actions).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              Recurso
              <select
                className="form-control"
                value={resource}
                onChange={(event) => {
                  setResource(event.target.value)
                  resetPage()
                }}
              >
                <option value="">Todos</option>
                {Object.entries(resources).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
          </>
        }
      />
      {query.isError ? (
        <button
          className="button-secondary mt-4"
          onClick={() => void query.refetch()}
        >
          Tentar novamente
        </button>
      ) : null}
    </>
  )
}
