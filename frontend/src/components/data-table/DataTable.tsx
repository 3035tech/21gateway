import { useId, useState } from 'react'
import type { ReactNode } from 'react'
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import type {
  ColumnDef,
  OnChangeFn,
  PaginationState,
  SortingState,
} from '@tanstack/react-table'
import { PAGE_SIZE_OPTIONS } from './constants'
import { useTranslation } from 'react-i18next'
import { AppSelect } from '../admin/AppSelect'
import {
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  Search,
  LoaderCircle,
  SearchX,
  Inbox,
  CloudOff,
  ArrowLeftRight,
} from 'lucide-react'

type DataTableProps<T> = {
  columns: ColumnDef<T>[]
  data: T[]
  total: number
  getRowId: (row: T) => string
  pagination: PaginationState
  onPaginationChange: OnChangeFn<PaginationState>
  sorting: SortingState
  onSortingChange: OnChangeFn<SortingState>
  search: string
  onSearchChange: (value: string) => void
  caption?: string
  searchPlaceholder?: string
  wide?: boolean
  filters?: ReactNode
  isLoading?: boolean
  error?: string
  hasActiveFilters?: boolean
  onClearFilters?: () => void
  emptyAction?: ReactNode
}

export function DataTable<T>({
  columns,
  data,
  total,
  getRowId,
  pagination,
  onPaginationChange,
  sorting,
  onSortingChange,
  search,
  onSearchChange,
  filters,
  caption,
  searchPlaceholder,
  wide = false,
  isLoading = false,
  error,
  hasActiveFilters = false,
  onClearFilters,
  emptyAction,
}: DataTableProps<T>) {
  const { t } = useTranslation()
  const tableCaption = caption ?? t('table.caption')
  const id = useId()
  const [draft, setDraft] = useState(search)
  const isFiltered = !!search.trim() || hasActiveFilters
  // eslint-disable-next-line react-hooks/incompatible-library -- React Compiler is not enabled; this table is not memoized.
  const table = useReactTable({
    columns,
    data,
    getRowId,
    rowCount: error ? 0 : total,
    state: { pagination, sorting },
    onPaginationChange,
    onSortingChange,
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
    enableMultiSort: false,
    getCoreRowModel: getCoreRowModel(),
  })
  return (
    <section className="space-y-5" aria-busy={isLoading}>
      <form
        className="flex flex-wrap items-end gap-3 rounded-xl border border-slate-200 bg-white p-4 sm:p-5"
        onSubmit={(event) => {
          event.preventDefault()
          onSearchChange(draft.trim())
        }}
      >
        <div className="flex w-full flex-col gap-1 sm:min-w-48 sm:flex-1">
          <label htmlFor={`${id}-search`} className="text-sm font-semibold">
            {t('common.search')}
          </label>
          <div className="relative">
            <Search
              size={17}
              aria-hidden="true"
              className="pointer-events-none absolute left-3 top-3.5 text-slate-400"
            />
            <input
              id={`${id}-search`}
              type="search"
              placeholder={searchPlaceholder ?? t('table.searchPlaceholder')}
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              className="form-control w-full pl-10"
            />
          </div>
        </div>
        {filters}
        <button type="submit" className="button-primary">
          {t('common.search')}
        </button>
      </form>
      {error ? (
        <p role="alert" className="rounded-md bg-red-50 p-4 text-red-800">
          {error}
        </p>
      ) : null}
      <p
        className={`flex items-center gap-2 text-xs text-slate-500 ${wide ? '' : 'sm:hidden'}`}
      >
        <ArrowLeftRight size={15} aria-hidden="true" /> {t('table.scrollHint')}
      </p>
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div
          className="overflow-x-auto"
          role="region"
          aria-label={t('table.regionLabel', { caption: tableCaption })}
          tabIndex={0}
        >
          <table className="data-grid w-full min-w-max border-collapse text-left text-sm">
            <caption className="sr-only">{tableCaption}</caption>
            <thead className="border-b border-slate-200 bg-slate-50 text-slate-700">
              {table.getHeaderGroups().map((group) => (
                <tr key={group.id}>
                  {group.headers.map((header) => {
                    const direction = header.column.getIsSorted()
                    return (
                      <th
                        key={header.id}
                        scope="col"
                        aria-sort={
                          direction === 'asc'
                            ? 'ascending'
                            : direction === 'desc'
                              ? 'descending'
                              : 'none'
                        }
                        className={`whitespace-nowrap px-5 py-3 font-semibold ${header.column.id === 'actions' ? 'text-center' : ''}`}
                      >
                        {header.isPlaceholder ? null : header.column.getCanSort() ? (
                          <button
                            type="button"
                            onClick={header.column.getToggleSortingHandler()}
                            className="button-sort"
                          >
                            {flexRender(
                              header.column.columnDef.header,
                              header.getContext(),
                            )}
                            {direction === 'asc' ? (
                              <ArrowUp
                                size={15}
                                aria-hidden="true"
                                className="text-brand-purple"
                              />
                            ) : direction === 'desc' ? (
                              <ArrowDown
                                size={15}
                                aria-hidden="true"
                                className="text-brand-purple"
                              />
                            ) : (
                              <ArrowUpDown
                                size={15}
                                aria-hidden="true"
                                className="text-slate-400"
                              />
                            )}
                          </button>
                        ) : (
                          flexRender(
                            header.column.columnDef.header,
                            header.getContext(),
                          )
                        )}
                      </th>
                    )
                  })}
                </tr>
              ))}
            </thead>
            <tbody>
              {!isLoading && !error
                ? table.getRowModel().rows.map((row) => (
                    <tr
                      key={row.id}
                      className="border-t border-slate-100 hover:bg-slate-50"
                    >
                      {row.getVisibleCells().map((cell) => (
                        <td
                          key={cell.id}
                          className={`px-5 py-3.5 align-middle text-slate-700 ${cell.column.id === 'actions' ? 'data-grid-actions text-center' : ''}`}
                        >
                          {cell.column.id === 'actions' ? (
                            <div className="flex justify-center">
                              {flexRender(
                                cell.column.columnDef.cell,
                                cell.getContext(),
                              )}
                            </div>
                          ) : (
                            flexRender(
                              cell.column.columnDef.cell,
                              cell.getContext(),
                            )
                          )}
                        </td>
                      ))}
                    </tr>
                  ))
                : null}
            </tbody>
          </table>
        </div>
        {isLoading || error || !table.getRowModel().rows.length ? (
          <div
            className="px-6 py-12 text-center text-sm text-slate-500"
            role={isLoading ? 'status' : undefined}
          >
            {isLoading ? (
              <LoaderCircle
                size={24}
                aria-hidden="true"
                className="mx-auto mb-3 animate-spin text-brand-purple"
              />
            ) : error ? (
              <CloudOff
                size={28}
                aria-hidden="true"
                className="mx-auto mb-3 text-slate-400"
              />
            ) : isFiltered ? (
              <SearchX
                size={28}
                aria-hidden="true"
                className="mx-auto mb-3 text-slate-400"
              />
            ) : (
              <Inbox
                size={28}
                aria-hidden="true"
                className="mx-auto mb-3 text-slate-400"
              />
            )}
            <p className="font-semibold text-slate-700">
              {isLoading
                ? t('table.loading')
                : error
                  ? t('table.unavailableHint')
                  : t(
                      isFiltered ? 'table.emptyFiltered' : 'table.emptyInitial',
                    )}
            </p>
            {!isLoading && !error ? (
              isFiltered && onClearFilters ? (
                <button
                  type="button"
                  className="button-secondary mt-4"
                  onClick={() => {
                    setDraft('')
                    onClearFilters()
                  }}
                >
                  {t('table.clearFilters')}
                </button>
              ) : !isFiltered && emptyAction ? (
                <div className="mt-4">{emptyAction}</div>
              ) : null
            ) : null}
          </div>
        ) : null}
      </div>
      {total > 0 && !isLoading && !error ? (
        <div className="flex flex-wrap items-center justify-between gap-4 px-1 text-sm text-slate-600">
          <label className="flex items-center gap-2 whitespace-nowrap">
            {t('common.itemsPerPage')}
            <AppSelect
              value={pagination.pageSize}
              label={t('common.itemsPerPage')}
              onValueChange={(value) =>
                onPaginationChange({
                  pageIndex: 0,
                  pageSize: Number(value),
                })
              }
              disabled={isLoading}
              options={PAGE_SIZE_OPTIONS.map((size) => ({
                value: size,
                label: String(size),
              }))}
            />
          </label>
          <div className="flex flex-wrap items-center gap-3">
            <span aria-live="polite">
              {total > 0 && !error && !isLoading ? (
                <span className="mr-3 hidden tabular-nums xl:inline">
                  {t('common.resultsRange', {
                    from: pagination.pageIndex * pagination.pageSize + 1,
                    to:
                      pagination.pageIndex * pagination.pageSize + data.length,
                    total,
                  })}
                </span>
              ) : null}
              {t('common.page', {
                page: error ? 1 : pagination.pageIndex + 1,
                pages: Math.max(1, table.getPageCount()),
                count: error ? 0 : total,
              })}
            </span>
            <button
              type="button"
              disabled={isLoading || !table.getCanPreviousPage()}
              onClick={() => table.previousPage()}
              aria-label={t('common.previous')}
              className="button-secondary"
            >
              {t('common.previous')}
            </button>
            <button
              type="button"
              disabled={isLoading || !table.getCanNextPage()}
              onClick={() => table.nextPage()}
              aria-label={t('common.next')}
              className="button-secondary"
            >
              {t('common.next')}
            </button>
          </div>
        </div>
      ) : null}
    </section>
  )
}
