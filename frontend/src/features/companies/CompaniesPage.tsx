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
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { ArrowUpRight } from 'lucide-react'
export function CompaniesPage() {
  const { t } = useTranslation()
  const user = useUser()
  const writable = user?.role !== 'viewer'
  const [editor, setEditor] = useState<Company | null | undefined>()
  const [archive, setArchive] = useState<Company | null>(null)
  const mutation = useAdminMutation(archiveCompany, () => setArchive(null))
  const columns: ColumnDef<Company>[] = [
    { accessorKey: 'id', header: 'ID' },
    {
      accessorKey: 'name',
      header: t('common.company'),
      cell: ({ row }) => (
        <Link
          className="inline-flex items-center gap-1 font-semibold text-brand-purple hover:underline"
          to={`/empresas/${row.original.id}`}
        >
          {row.original.name}
          <ArrowUpRight size={14} aria-hidden="true" />
        </Link>
      ),
    },
    { accessorKey: 'reference', header: t('common.reference') },
    {
      accessorKey: 'active',
      header: t('common.status'),
      cell: ({ row }) => (
        <StatusBadge status={row.original.active ? 'active' : 'blocked'} />
      ),
    },
    {
      accessorKey: 'createdAt',
      header: t('common.createdAt'),
      cell: ({ row }) => <DateCell value={row.original.createdAt} />,
    },
  ]
  columns.push({
    id: 'actions',
    header: t('common.actions'),
    enableSorting: false,
    cell: ({ row }) => (
      <div className="flex gap-1">
        <Link className="table-action" to={`/empresas/${row.original.id}`}>
          {t('common.view')}
        </Link>
        {writable ? (
          <>
            <button
              className="table-action"
              onClick={() => setEditor(row.original)}
            >
              {t('common.edit')}
            </button>
            <button
              className="table-action-danger"
              onClick={() => setArchive(row.original)}
            >
              {t('common.archive')}
            </button>
          </>
        ) : null}
      </div>
    ),
  })
  return (
    <>
      <Listing
        title={t('nav.companies')}
        description={t('companies.description')}
        searchPlaceholder={t('companies.searchPlaceholder')}
        queryKey="companies"
        columns={columns}
        load={listCompanies}
        onCreate={writable ? () => setEditor(null) : undefined}
        createLabel={t('companies.new')}
        filterName="active"
        filterOptions={[
          ['true', t('companies.activePlural')],
          ['false', t('companies.blockedPlural')],
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
          title={t('companies.archiveTitle')}
          description={t('companies.archiveDescription', {
            name: archive.name,
          })}
          pending={mutation.isPending}
          error={mutation.error}
          onConfirm={() => mutation.mutate(archive.id)}
          onClose={() => setArchive(null)}
        />
      ) : null}
    </>
  )
}
