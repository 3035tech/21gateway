import { DateCell } from '../../components/admin/DateCell'
import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { Confirmation } from '../../components/admin/Confirmation'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useUser } from '../auth/context'
import { errorMessage, roleLabels } from '../../lib/admin'
import { listUsers, archiveUser, resendInvitation } from './api'
import type { SystemUser } from './api'
import { UserForm } from './UserForm'
export function UsersPage() {
  const user = useUser()
  const writable = user?.role === 'admin'
  const [editor, setEditor] = useState<SystemUser | null | undefined>()
  const [archive, setArchive] = useState<SystemUser | null>(null)
  const mutation = useAdminMutation(archiveUser, () => setArchive(null))
  const [inviteSent, setInviteSent] = useState(false)
  const resend = useMutation({
    mutationFn: resendInvitation,
    onSuccess: () => setInviteSent(true),
  })
  const columns: ColumnDef<SystemUser>[] = [
    { accessorKey: 'id', header: 'ID' },
    { accessorKey: 'name', header: 'Nome' },
    { accessorKey: 'email', header: 'E-mail' },
    {
      accessorKey: 'role',
      header: 'Perfil',
      cell: ({ row }) => roleLabels[row.original.role],
    },
    {
      accessorKey: 'active',
      header: 'Status',
      cell: ({ row }) => (
        <StatusBadge status={row.original.active ? 'active' : 'blocked'} />
      ),
    },
    {
      accessorKey: 'createdAt',
      header: 'Cadastro',
      cell: ({ row }) => <DateCell value={row.original.createdAt} />,
    },
  ]
  if (writable)
    columns.push({
      id: 'actions',
      header: 'Ações',
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex gap-1">
          <button
            className="table-action"
            onClick={() => setEditor(row.original)}
          >
            Editar
          </button>
          <button
            className="table-action"
            disabled={resend.isPending}
            onClick={() => {
              setInviteSent(false)
              resend.mutate(row.original.id)
            }}
          >
            Reenviar convite
          </button>
          <button
            className="table-action-danger"
            onClick={() => setArchive(row.original)}
          >
            Arquivar
          </button>
        </div>
      ),
    })
  return (
    <>
      {inviteSent ? (
        <p
          role="status"
          className="mb-4 rounded-lg bg-emerald-50 p-3 text-sm text-emerald-800"
        >
          Convite enviado. O novo link expira em 15 minutos.
        </p>
      ) : null}
      {resend.isError ? (
        <p
          role="alert"
          className="mb-4 rounded-lg bg-rose-50 p-3 text-sm text-rose-700"
        >
          {errorMessage(resend.error)}
        </p>
      ) : null}
      <Listing
        title="Usuários"
        description="Gerencie os acessos internos ao backoffice. Novos usuários definem a própria senha pelo convite enviado por e-mail."
        searchPlaceholder="Nome ou e-mail do usuário"
        queryKey="users"
        columns={columns}
        load={listUsers}
        onCreate={writable ? () => setEditor(null) : undefined}
        createLabel="Novo usuário"
        filterName="active"
        filterOptions={[
          ['true', 'Ativos'],
          ['false', 'Bloqueados'],
        ]}
      />
      {editor !== undefined ? (
        <UserForm
          user={editor ?? undefined}
          onClose={() => setEditor(undefined)}
        />
      ) : null}
      {archive ? (
        <Confirmation
          title="Arquivar usuário"
          description={`Arquivar ${archive.name} encerrará suas sessões de acesso. O histórico será mantido.`}
          pending={mutation.isPending}
          error={mutation.error}
          onConfirm={() => mutation.mutate(archive.id)}
          onClose={() => setArchive(null)}
        />
      ) : null}
    </>
  )
}
