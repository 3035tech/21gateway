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
import { Archive, ArrowUpRight, Eye, Pencil } from 'lucide-react'
import {
  TableIconButton,
  TableIconLink,
} from '../../components/admin/TableIconAction'
export function CompaniesPage() {
  const { t } = useTranslation()
  const user = useUser()
  const writable = user?.role !== 'viewer'
  const canCreate = user?.role === 'admin'
  const [editor, setEditor] = useState<Company | null | undefined>()
  const [archive, setArchive] = useState<Company | null>(null)
  const mutation = useAdminMutation(
    archiveCompany,
    () => setArchive(null),
    'feedback.archived',
  )
  const columns: ColumnDef<Company>[] = [
    {
      accessorKey: 'name',
      header: t('common.company'),
      cell: ({ row }) => (
        <Link
          className="inline-flex items-center gap-1 font-semibold text-brand-purple hover:underline"
          to={`/empresas/${row.original.publicRef}`}
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
      <div className="flex w-max items-center gap-1">
        <TableIconLink
          icon={Eye}
          label={t('common.view')}
          to={`/empresas/${row.original.publicRef}`}
        />
        {writable ? (
          <>
            <TableIconButton
              icon={Pencil}
              label={t('common.edit')}
              onClick={() => setEditor(row.original)}
            />
            <TableIconButton
              icon={Archive}
              label={t('common.archive')}
              danger
              onClick={() => setArchive(row.original)}
            />
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
        onCreate={canCreate ? () => setEditor(null) : undefined}
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
