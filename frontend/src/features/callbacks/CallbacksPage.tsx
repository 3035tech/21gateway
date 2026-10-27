import { DateCell } from '../../components/admin/DateCell'
import { useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { Confirmation } from '../../components/admin/Confirmation'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useUser } from '../auth/context'
import { listCallbacks, retryCallback } from './api'
import type { Callback } from './api'
const callbackErrors: Record<string, string> = {
  NETWORK_ERROR: 'Não foi possível conectar ao destino',
  HTTP_ERROR: 'O destino recusou a notificação',
  ACCESS_BLOCKED: 'Entrega bloqueada pela configuração da conta',
}
export function CallbacksPage() {
  const writable = useUser()?.role !== 'viewer'
  const [retry, setRetry] = useState<Callback | null>(null)
  const mutation = useAdminMutation(retryCallback, () => setRetry(null))
  const columns: ColumnDef<Callback>[] = [
    { accessorKey: 'id', header: 'ID' },
    { accessorKey: 'attemptId', header: 'Envio' },
    {
      accessorKey: 'companyName',
      header: 'Cliente',
      cell: ({ row }) =>
        row.original.companyName ||
        (row.original.companyId
          ? `Empresa #${row.original.companyId}`
          : 'Sem empresa vinculada'),
    },
    { accessorKey: 'url', header: 'Destino' },
    {
      accessorKey: 'status',
      header: 'Entrega do SMS',
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      accessorKey: 'state',
      header: 'Callback',
      cell: ({ row }) => <StatusBadge status={row.original.state} />,
    },
    { accessorKey: 'attempts', header: 'Tentativas' },
    {
      accessorKey: 'lastError',
      header: 'Última falha',
      cell: ({ row }) =>
        row.original.lastError ? (
          <span title={row.original.lastError}>
            {callbackErrors[row.original.lastError] ?? row.original.lastError}
          </span>
        ) : (
          'Sem falhas registradas'
        ),
    },
    {
      accessorKey: 'createdAt',
      header: 'Data',
      cell: ({ row }) => <DateCell value={row.original.createdAt} />,
    },
  ]
  if (writable)
    columns.push({
      id: 'actions',
      header: 'Ações',
      enableSorting: false,
      cell: ({ row }) =>
        row.original.state === 'failed' ? (
          <button
            className="table-action"
            onClick={() => {
              mutation.reset()
              setRetry(row.original)
            }}
          >
            Tentar novamente
          </button>
        ) : null,
    })
  return (
    <>
      <Listing
        title="Notificações de entrega"
        description="Acompanhe os status DLR encaminhados aos clientes. A integração com o fornecedor ainda precisa de homologação."
        searchPlaceholder="URL de destino"
        queryKey="callbacks"
        columns={columns}
        load={listCallbacks}
        filterName="callbackState"
        filterOptions={[
          ['pending', 'Pendentes'],
          ['processing', 'Processando'],
          ['delivered', 'Entregues'],
          ['failed', 'Falhos'],
        ]}
      />
      {retry ? (
        <Confirmation
          title="Reprocessar callback"
          description={`A notificação do envio ${retry.attemptId} voltará à fila. Isso não reenvia o SMS.`}
          pending={mutation.isPending}
          error={mutation.error}
          onConfirm={() => mutation.mutate(retry.id)}
          onClose={() => setRetry(null)}
        />
      ) : null}
    </>
  )
}
