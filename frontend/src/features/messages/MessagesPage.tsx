import { DateCell } from '../../components/admin/DateCell'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { listMessages } from './api'
import type { Message } from './api'
const columns: ColumnDef<Message>[] = [
  { accessorKey: 'id', header: 'ID do envio' },
  {
    accessorKey: 'companyName',
    header: 'Cliente',
    cell: ({ row }) =>
      row.original.companyName ||
      (row.original.companyId
        ? `Empresa #${row.original.companyId}`
        : 'Sem empresa vinculada'),
  },
  { accessorKey: 'destination', header: 'Destinatário' },
  { accessorKey: 'correlationId', header: 'Referência' },
  {
    accessorKey: 'state',
    header: 'Envio',
    cell: ({ row }) => <StatusBadge status={row.original.state} />,
  },
  {
    accessorKey: 'deliveryStatus',
    header: 'Entrega',
    cell: ({ row }) =>
      row.original.deliveryStatus ? (
        <StatusBadge status={row.original.deliveryStatus} />
      ) : (
        'Sem confirmação'
      ),
  },
  {
    accessorKey: 'failureCode',
    header: 'Código da falha',
  },
  {
    accessorKey: 'createdAt',
    header: 'Data',
    cell: ({ row }) => <DateCell value={row.original.createdAt} />,
  },
]
export function MessagesPage() {
  return (
    <Listing
      title="Envios"
      description="Aceite pelo fornecedor e entrega são etapas diferentes. Não reenvie automaticamente resultados incertos."
      searchPlaceholder="Destinatário ou referência"
      queryKey="messages"
      columns={columns}
      load={listMessages}
      filterName="state"
      filterOptions={[
        ['pending', 'Pendentes'],
        ['accepted', 'Aceitos'],
        ['rejected', 'Rejeitados'],
        ['unknown', 'Resultados incertos'],
      ]}
    />
  )
}
