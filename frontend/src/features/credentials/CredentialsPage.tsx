import { DateCell } from '../../components/admin/DateCell'
import { useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { Confirmation } from '../../components/admin/Confirmation'
import { Dialog } from '../../components/admin/Dialog'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useUser } from '../auth/context'
import { listCredentials, archiveCredential, rotateCredential } from './api'
import type { Credential } from './api'
import { CredentialForm } from './CredentialForm'
export function CredentialsPage() {
  const writable = useUser()?.role !== 'viewer'
  const [editor, setEditor] = useState<Credential | null | undefined>()
  const [action, setAction] = useState<{
    credential: Credential
    rotate: boolean
  } | null>(null)
  const [secret, setSecret] = useState<{
    username: string
    password: string
  } | null>(null)
  const mutation = useAdminMutation<{
    credential: Credential
    rotate: boolean
  }>(
    async (value) => {
      if (value.rotate) {
        const result = await rotateCredential(value.credential.id)
        setSecret({
          username: result.credential.username,
          password: result.password,
        })
      } else await archiveCredential(value.credential.id)
    },
    () => setAction(null),
  )
  const columns: ColumnDef<Credential>[] = [
    { accessorKey: 'id', header: 'ID' },
    { accessorKey: 'username', header: 'Usuário da API' },
    { accessorKey: 'companyName', header: 'Empresa' },
    {
      accessorKey: 'providerTokenConfigured',
      header: 'Zoug',
      cell: ({ row }) =>
        row.original.providerTokenConfigured
          ? 'Token configurado'
          : 'Sem token',
    },
    {
      accessorKey: 'whitelistEnabled',
      header: 'Acesso por IP',
      cell: ({ row }) =>
        row.original.whitelistEnabled
          ? `${row.original.allowedIPs.length} IPs/redes`
          : 'Sem restrição',
    },
    {
      accessorKey: 'active',
      header: 'Status',
      cell: ({ row }) => (
        <StatusBadge status={row.original.active ? 'active' : 'blocked'} />
      ),
    },
    {
      accessorKey: 'callbacksEnabled',
      id: 'notifications',
      header: 'Notificações',
      cell: ({ row }) => (
        <div className="flex flex-wrap gap-1.5">
          {[
            ['DLR', row.original.callbacksEnabled],
            ['MO', row.original.moCallbacksEnabled],
          ].map(([label, enabled]) => (
            <span
              key={String(label)}
              className={`inline-flex rounded-md px-2 py-1 text-xs font-semibold ${enabled ? 'bg-purple-50 text-brand-purple' : 'bg-slate-100 text-slate-500'}`}
              title={enabled ? `${label} habilitado` : `${label} desabilitado`}
            >
              {label}{' '}
              <span className="sr-only">
                {enabled ? 'habilitado' : 'desabilitado'}
              </span>
            </span>
          ))}
        </div>
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
        <div className="flex flex-nowrap gap-1">
          <button
            className="table-action"
            onClick={() => setEditor(row.original)}
          >
            Editar
          </button>
          <button
            className="table-action"
            onClick={() => {
              mutation.reset()
              setAction({ credential: row.original, rotate: true })
            }}
          >
            Trocar senha
          </button>
          <button
            className="table-action-danger"
            onClick={() => {
              mutation.reset()
              setAction({ credential: row.original, rotate: false })
            }}
          >
            Arquivar
          </button>
        </div>
      ),
    })
  return (
    <>
      <Listing
        title="Credenciais da API"
        description="Configure o acesso dos clientes, os tokens de envio e as notificações de cada conta."
        searchPlaceholder="Usuário da API"
        queryKey="credentials"
        columns={columns}
        load={listCredentials}
        onCreate={writable ? () => setEditor(null) : undefined}
        createLabel="Nova credencial"
        filterName="active"
        filterOptions={[
          ['true', 'Ativas'],
          ['false', 'Bloqueadas'],
        ]}
      />
      {editor !== undefined ? (
        <CredentialForm
          credential={editor ?? undefined}
          onClose={() => setEditor(undefined)}
          onSecret={(username, password) => setSecret({ username, password })}
        />
      ) : null}
      {action ? (
        <Confirmation
          title={action.rotate ? 'Trocar senha da API' : 'Arquivar credencial'}
          description={
            action.rotate
              ? `A senha atual de ${action.credential.username} deixará de funcionar. Atualize a integração do cliente após a troca.`
              : `A credencial ${action.credential.username} será bloqueada. O histórico será mantido.`
          }
          pending={mutation.isPending}
          error={mutation.error}
          onConfirm={() => mutation.mutate(action)}
          onClose={() => setAction(null)}
        />
      ) : null}
      {secret && editor === undefined && action === null ? (
        <Dialog title="Guarde a nova senha" onClose={() => setSecret(null)}>
          <p className="mb-4 text-sm text-slate-600">
            A senha só aparece agora. Guarde-a em um local seguro antes de
            fechar.
          </p>
          <label className="field">
            Usuário
            <input readOnly className="form-control" value={secret.username} />
          </label>
          <label className="field mt-4">
            Senha
            <input
              readOnly
              autoComplete="off"
              className="form-control font-mono"
              value={secret.password}
            />
          </label>
          <button
            className="button-primary mt-6 w-full"
            onClick={() => setSecret(null)}
          >
            Já guardei a senha
          </button>
        </Dialog>
      ) : null}
    </>
  )
}
