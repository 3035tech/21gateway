import { useState } from 'react'
import type { ReactNode } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { DataTable } from '../data-table/DataTable'
import { useListState } from '../data-table/useListState'
import type { ListParams, Page } from '../../lib/admin'
import { errorMessage } from '../../lib/admin'
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
}: Props<T>) {
  const state = useListState()
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
    ...(company ? { companyId: Number(company) } : {}),
  }
  const query = useQuery({
    queryKey: ['admin', queryKey, params],
    queryFn: ({ signal }) => load(params, signal),
  })
  return (
    <>
      <div className="mb-7 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
            {title}
          </h1>
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
        searchPlaceholder={searchPlaceholder}
        columns={columns}
        data={query.data?.items ?? []}
        total={query.data?.total ?? 0}
        getRowId={(row) => String(row.id)}
        isLoading={query.isPending}
        error={query.isError ? errorMessage(query.error) : undefined}
        filters={
          <>
            {filterName ? (
              <label className="flex flex-col gap-1 text-sm font-semibold">
                Status
                <select
                  className="form-control"
                  value={filter}
                  onChange={(e) => {
                    setFilter(e.target.value)
                    state.onPaginationChange({
                      ...state.pagination,
                      pageIndex: 0,
                    })
                  }}
                >
                  <option value="">Todos</option>
                  {filterOptions?.map(([value, label]) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </label>
            ) : null}
            {['credentials', 'messages', 'callbacks'].includes(queryKey) ? (
              <label className="flex flex-col gap-1 text-sm font-semibold">
                ID da empresa
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
          Tentar novamente
        </button>
      ) : null}
    </>
  )
}
