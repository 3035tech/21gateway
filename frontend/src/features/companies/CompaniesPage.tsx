import { DateCell } from '../../components/admin/DateCell'
import { useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { Confirmation } from '../../components/admin/Confirmation'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useUser } from '../auth/context'
import { listCompanies, archiveCompany } from './api'
import type { Company } from './api'
import { CompanyForm } from './CompanyForm'
export function CompaniesPage() {
  const user = useUser()
  const writable = user?.role !== 'viewer'
  const [editor, setEditor] = useState<Company | null | undefined>()
  const [archive, setArchive] = useState<Company | null>(null)
  const mutation = useAdminMutation(archiveCompany, () => setArchive(null))
  const columns: ColumnDef<Company>[] = [
    { accessorKey: 'id', header: 'ID' },
    { accessorKey: 'name', header: 'Empresa' },
    { accessorKey: 'reference', header: 'Referência' },
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
      <Listing
        title="Empresas"
        description="Organize os clientes que utilizam a API de SMS."
        searchPlaceholder="Nome ou referência da empresa"
        queryKey="companies"
        columns={columns}
        load={listCompanies}
        onCreate={writable ? () => setEditor(null) : undefined}
        createLabel="Nova empresa"
        filterName="active"
        filterOptions={[
          ['true', 'Ativas'],
          ['false', 'Bloqueadas'],
        ]}
      />
      {editor !== undefined ? (
        <CompanyForm
          company={editor ?? undefined}
          onClose={() => setEditor(undefined)}
        />
      ) : null}
      {archive ? (
        <Confirmation
          title="Arquivar empresa"
          description={`Arquivar ${archive.name} bloqueará seus acessos de API. O histórico será mantido.`}
          pending={mutation.isPending}
          error={mutation.error}
          onConfirm={() => mutation.mutate(archive.id)}
          onClose={() => setArchive(null)}
        />
      ) : null}
    </>
  )
}
