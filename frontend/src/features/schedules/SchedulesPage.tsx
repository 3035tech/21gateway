import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { DataTable } from '../../components/data-table/DataTable'
import { useListState } from '../../components/data-table/useListState'
import { DateCell } from '../../components/admin/DateCell'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { errorMessage } from '../../lib/admin'
import { listSchedules, scheduleStates } from './api'
import type { Schedule, ScheduleListParams } from './api'

const columns: ColumnDef<Schedule>[] = [
  { accessorKey: 'attemptId', header: 'ID do envio' },
  {
    accessorKey: 'companyName',
    header: 'Cliente',
    cell: ({ row }) => row.original.companyName || 'Sem empresa vinculada',
  },
  { accessorKey: 'username', header: 'Usuário da API' },
  {
    accessorKey: 'createdAt',
    header: 'Solicitado em',
    cell: ({ row }) => <DateCell value={row.original.createdAt} />,
  },
  {
    accessorKey: 'scheduledAt',
    header: 'Agendado para',
    cell: ({ row }) => <DateCell value={row.original.scheduledAt} />,
  },
  {
    accessorKey: 'state',
    header: 'Fila',
    cell: ({ row }) => <StatusBadge status={row.original.state} />,
  },
  {
    accessorKey: 'submissionState',
    header: 'Envio',
    cell: ({ row }) => <StatusBadge status={row.original.submissionState} />,
  },
  { accessorKey: 'destination', header: 'Destinatário' },
  { accessorKey: 'correlationId', header: 'Referência' },
  {
    accessorKey: 'dispatchedAt',
    header: 'Despachado em',
    cell: ({ row }) =>
      row.original.dispatchedAt ? (
        <DateCell value={row.original.dispatchedAt} />
      ) : (
        'Ainda não despachado'
      ),
  },
]

const stateLabels: Record<(typeof scheduleStates)[number], string> = {
  queued: 'Na fila',
  claimed: 'Reservado',
  dispatching: 'Em envio',
  done: 'Aceito pelo fornecedor',
  failed: 'Falhou',
  unknown: 'Resultado incerto',
}

function utc(value: string) {
  if (!value) return undefined
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString()
}

export function SchedulesPage() {
  const table = useListState()
  const [companyId, setCompanyId] = useState('')
  const [credentialId, setCredentialId] = useState('')
  const [state, setState] = useState('')
  const [dueFrom, setDueFrom] = useState('')
  const [dueTo, setDueTo] = useState('')
  const invalidRange = !!dueFrom && !!dueTo && dueFrom > dueTo
  const sort = table.sorting[0]
  const params: ScheduleListParams = {
    page: table.pagination.pageIndex + 1,
    pageSize: table.pagination.pageSize,
    search: table.search,
    sort: sort?.id,
    direction: sort?.desc ? 'desc' : 'asc',
    companyId: companyId ? Number(companyId) : undefined,
    credentialId: credentialId ? Number(credentialId) : undefined,
    state: state || undefined,
    dueFrom: utc(dueFrom),
    dueTo: utc(dueTo),
  }
  const query = useQuery({
    queryKey: ['admin', 'schedules', params],
    queryFn: ({ signal }) => listSchedules(params, signal),
    enabled: !invalidRange,
  })
  const resetPage = () =>
    table.onPaginationChange({ ...table.pagination, pageIndex: 0 })
  return (
    <>
      <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
        Agendamentos
      </h1>
      <p className="mb-2 mt-2 max-w-3xl text-sm leading-6 text-slate-600">
        Veja quando o cliente solicitou o agendamento, o horário previsto e o
        andamento da fila. O usuário da API identifica a credencial técnica;
        esta integração não identifica a pessoa que fez a solicitação.
      </p>
      <p className="mb-6 text-xs text-slate-500">
        Os horários e os filtros usam o fuso do seu navegador. Aceite pelo
        fornecedor ainda não confirma entrega.
      </p>
      {invalidRange ? (
        <p role="alert" className="mb-4 text-sm text-rose-700">
          O início do período deve ser anterior ao fim.
        </p>
      ) : null}
      <DataTable
        {...table}
        caption="Agendamentos de SMS"
        wide
        searchPlaceholder="Destinatário ou referência"
        columns={columns}
        data={invalidRange ? [] : (query.data?.items ?? [])}
        total={invalidRange ? 0 : (query.data?.total ?? 0)}
        getRowId={(row) => String(row.id)}
        isLoading={!invalidRange && query.isPending}
        error={query.isError ? errorMessage(query.error) : undefined}
        filters={
          <>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              Empresa ID
              <input
                className="form-control w-28"
                type="number"
                min={1}
                value={companyId}
                onChange={(event) => {
                  setCompanyId(event.target.value)
                  resetPage()
                }}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              Credencial ID
              <input
                className="form-control w-28"
                type="number"
                min={1}
                value={credentialId}
                onChange={(event) => {
                  setCredentialId(event.target.value)
                  resetPage()
                }}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              Estado da fila
              <select
                className="form-control"
                value={state}
                onChange={(event) => {
                  setState(event.target.value)
                  resetPage()
                }}
              >
                <option value="">Todos</option>
                {scheduleStates.map((value) => (
                  <option key={value} value={value}>
                    {stateLabels[value]}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              De
              <input
                className="form-control"
                type="datetime-local"
                value={dueFrom}
                onChange={(event) => {
                  setDueFrom(event.target.value)
                  resetPage()
                }}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              Até
              <input
                className="form-control"
                type="datetime-local"
                value={dueTo}
                onChange={(event) => {
                  setDueTo(event.target.value)
                  resetPage()
                }}
              />
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
