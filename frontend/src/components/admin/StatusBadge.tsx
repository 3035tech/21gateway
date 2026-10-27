const labels: Record<string, string> = {
  active: 'Ativo',
  blocked: 'Bloqueado',
  pending: 'Pendente',
  queued: 'Na fila',
  claimed: 'Reservado',
  dispatching: 'Em envio',
  done: 'Aceito pelo fornecedor',
  accepted: 'Aceito',
  rejected: 'Rejeitado',
  unknown: 'Resultado incerto',
  processing: 'Processando',
  delivered: 'Entregue',
  failed: 'Falhou',
  SENT: 'Enviado',
  DELIVERED: 'Entregue',
  NOT_DELIVERED: 'Não entregue',
  DUPLICATED: 'Duplicado',
  INVALID_DESTINATION_ADDRESS: 'Destino inválido',
  BLACKLIST: 'Lista de bloqueio',
  MESSAGE_CANCELLED: 'Cancelado',
  INVALID_MESSAGE_FORMAT: 'Formato inválido',
  GENERAL_ERROR: 'Erro de entrega',
}
export function StatusBadge({ status }: { status: string }) {
  const color = ['active', 'accepted', 'delivered', 'DELIVERED'].includes(
    status,
  )
    ? 'bg-emerald-50 text-emerald-800'
    : [
          'blocked',
          'rejected',
          'failed',
          'NOT_DELIVERED',
          'GENERAL_ERROR',
          'INVALID_DESTINATION_ADDRESS',
          'INVALID_MESSAGE_FORMAT',
          'BLACKLIST',
        ].includes(status)
      ? 'bg-rose-50 text-rose-800'
      : ['processing', 'claimed', 'dispatching', 'SENT'].includes(status)
        ? 'bg-sky-50 text-sky-800'
        : 'bg-amber-50 text-amber-800'
  return (
    <span
      className={`inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold ${color}`}
    >
      <span
        aria-hidden="true"
        className="h-1.5 w-1.5 rounded-full bg-current"
      />
      {labels[status] ?? status}
    </span>
  )
}
