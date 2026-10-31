import { useState } from 'react'
import type { ReactNode } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { useQuery } from '@tanstack/react-query'
import { Plus, RotateCcw } from 'lucide-react'
import { DataTable } from '../data-table/DataTable'
import { useListState } from '../data-table/useListState'
import type { ListParams, Page } from '../../lib/admin'
import { errorMessage } from '../../lib/admin'
import { useTranslation } from 'react-i18next'
import { AppSelect } from './AppSelect'
import { useCompanyView } from '../../features/companies/CompanyViewContext'
import { useLocation } from 'react-router-dom'
type Props<T extends { id: number }> = {
  title: string
  description: string
  searchPlaceholder?: string
  queryKey: string
  columns: ColumnDef<T>[]
  load: (params: ListParams, signal?: AbortSignal) => Promise<Page<T>>
  onCreate?: () => void
  createLabel?: string
  filterName?: 'active' | 'state' | 'callbackState'
  filterOptions?: [string, string][]
  extra?: ReactNode
  fixedCompanyId?: number
  headingLevel?: 1 | 2
  stickyActions?: boolean
}
export function Listing<T extends { id: number }>({
  title,
  description,
  searchPlaceholder,
  queryKey,
  columns,
  load,
  onCreate,
  createLabel,
  filterName,
  filterOptions,
  extra,
  fixedCompanyId,
  headingLevel = 1,
  stickyActions = false,
}: Props<T>) {
  const { t } = useTranslation()
  const selectedCompanyId = useCompanyView()
  const location = useLocation()
  fixedCompanyId ??= selectedCompanyId
  const state = useListState(
    [],
    new URLSearchParams(location.search).get('search') ?? '',
  )
  const [filter, setFilter] = useState('')
  const [company, setCompany] = useState('')
  const sort = state.sorting[0]
  const params: ListParams = {
    page: state.pagination.pageIndex + 1,
    pageSize: state.pagination.pageSize,
    search: state.search,
    sort: sort?.id,
    direction: sort?.desc ? 'desc' : 'asc',
    ...(filterName && filter
      ? { [filterName]: filterName === 'active' ? filter === 'true' : filter }
      : {}),
    ...(fixedCompanyId
      ? { companyId: fixedCompanyId }
      : company
        ? { companyId: Number(company) }
        : {}),
  }
  const query = useQuery({
    queryKey: ['admin', queryKey, params],
    queryFn: ({ signal }) => load(params, signal),
  })
  return (
    <>
      <div className="mb-7 flex flex-wrap items-start justify-between gap-4">
        <div>
          {headingLevel === 1 ? (
            <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
              {title}
            </h1>
          ) : (
            <h2 className="text-2xl font-bold tracking-tight sm:text-3xl">
              {title}
            </h2>
          )}
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-500">
            {description}
          </p>
        </div>
        {onCreate ? (
          <button className="button-primary" onClick={onCreate}>
            <Plus size={17} />
            {createLabel}
          </button>
        ) : null}
      </div>
      {extra}
      <DataTable
        {...state}
        caption={title}
        wide={columns.length >= 8}
        stickyActions={stickyActions}
        searchPlaceholder={searchPlaceholder}
        columns={columns}
        data={query.data?.items ?? []}
        total={query.data?.total ?? 0}
        getRowId={(row) => String(row.id)}
        isLoading={query.isPending}
        error={query.isError ? errorMessage(query.error) : undefined}
        hasActiveFilters={!!filter || !!company}
        onClearFilters={() => {
          setFilter('')
          setCompany('')
          state.onSearchChange('')
        }}
        emptyAction={
          onCreate ? (
            <button type="button" className="button-primary" onClick={onCreate}>
              <Plus size={17} aria-hidden="true" />
              {createLabel}
            </button>
          ) : undefined
        }
        filters={
          <>
            {filterName ? (
              <label className="flex flex-col gap-1 text-sm font-semibold">
                {t('common.status')}
                <AppSelect
                  value={filter}
                  label={t('common.status')}
                  onValueChange={(value) => {
                    setFilter(value)
                    state.onPaginationChange({
                      ...state.pagination,
                      pageIndex: 0,
                    })
                  }}
                  options={[
                    { value: '', label: t('common.all') },
                    ...(filterOptions?.map(([value, label]) => ({
                      value,
                      label,
                    })) ?? []),
                  ]}
                />
              </label>
            ) : null}
            {!fixedCompanyId &&
            [
              'credentials',
              'messages',
              'callbacks',
              'receivedCallbacks',
            ].includes(queryKey) ? (
              <label className="flex flex-col gap-1 text-sm font-semibold">
                {t('common.companyId')}
                <input
                  type="number"
                  min={1}
                  className="form-control w-32"
                  value={company}
                  onChange={(e) => {
                    setCompany(e.target.value)
                    state.onPaginationChange({
                      ...state.pagination,
                      pageIndex: 0,
                    })
                  }}
                />
              </label>
            ) : null}
          </>
        }
      />
      {query.isError ? (
        <button
          className="button-secondary mt-4"
          onClick={() => void query.refetch()}
        >
          <RotateCcw size={17} aria-hidden="true" />
          {t('common.retry')}
        </button>
      ) : null}
    </>
  )
}
