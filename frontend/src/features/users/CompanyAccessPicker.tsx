import { Check, RotateCcw, Search, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

type CompanyOption = { id: string; name: string }

export function CompanyAccessPicker({
  options,
  selected,
  selectedIds,
  selectedCount,
  selectedPage,
  selectedPageSize,
  onSelectedPageChange,
  search,
  onSearchChange,
  onToggle,
  page,
  pageSize,
  total,
  onPageChange,
  loading = false,
  error = false,
  onRetry,
}: {
  options: CompanyOption[]
  selected: CompanyOption[]
  selectedIds: string[]
  selectedCount: number
  selectedPage: number
  selectedPageSize: number
  onSelectedPageChange: (page: number) => void
  search: string
  onSearchChange: (value: string) => void
  onToggle: (id: string) => void
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
  loading?: boolean
  error?: boolean
  onRetry?: () => void
}) {
  const { t } = useTranslation()
  const selectedIdSet = new Set(selectedIds)
  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  const selectedPageCount = Math.max(
    1,
    Math.ceil(selectedCount / selectedPageSize),
  )

  return (
    <fieldset className="space-y-3">
      <legend className="text-sm font-semibold">
        {t('users.companyAccess')}
      </legend>
      <p className="text-xs text-slate-500">
        {t('users.selectedCompanies', { count: selectedCount })}
      </p>
      {selectedCount ? (
        <div className="space-y-2 rounded-xl border border-purple-100 bg-purple-50/40 p-3">
          <ul
            className="flex flex-wrap gap-2"
            aria-label={t('users.selectedCompanyList')}
          >
            {selected.map((company) => (
              <li key={company.id}>
                <button
                  type="button"
                  className="inline-flex min-h-9 max-w-full items-center gap-2 rounded-lg border border-purple-200 bg-white px-3 py-1 text-sm font-semibold text-brand-purple hover:bg-purple-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-purple"
                  aria-label={t('users.removeCompanyNamed', {
                    name: company.name,
                  })}
                  onClick={() => onToggle(company.id)}
                >
                  <span className="truncate">{company.name}</span>
                  <X size={15} aria-hidden="true" />
                </button>
              </li>
            ))}
          </ul>
          {selectedPageCount > 1 ? (
            <div className="flex items-center justify-between gap-2 text-xs text-slate-600">
              <button
                type="button"
                className="button-ghost min-h-9 px-2"
                disabled={selectedPage <= 1}
                onClick={() => onSelectedPageChange(selectedPage - 1)}
              >
                {t('common.previous')}
              </button>
              <span>
                {t('users.companyPage', {
                  page: selectedPage,
                  pages: selectedPageCount,
                })}
              </span>
              <button
                type="button"
                className="button-ghost min-h-9 px-2"
                disabled={selectedPage >= selectedPageCount}
                onClick={() => onSelectedPageChange(selectedPage + 1)}
              >
                {t('common.next')}
              </button>
            </div>
          ) : null}
        </div>
      ) : null}
      <label className="field">
        {t('users.findCompany')}
        <span className="relative block">
          <Search
            size={18}
            aria-hidden="true"
            className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-500"
          />
          <input
            className="form-control w-full pl-10"
            type="search"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={t('users.companySearchPlaceholder')}
          />
        </span>
      </label>
      <div className="overflow-hidden rounded-xl border border-slate-200">
        <div className="max-h-52 overflow-y-auto">
          {loading || error || !options.length ? (
            <div className="px-4 py-4 text-sm text-slate-500" role="status">
              <p>
                {error
                  ? t('users.companyLoadFailed')
                  : loading
                    ? t('common.loading')
                    : t('users.noCompanyMatches')}
              </p>
              {error && onRetry ? (
                <button
                  type="button"
                  className="button-ghost mt-2 min-h-9 px-2"
                  onClick={onRetry}
                >
                  <RotateCcw size={16} aria-hidden="true" />
                  {t('common.retry')}
                </button>
              ) : null}
            </div>
          ) : (
            options.map((company) => {
              const isSelected = selectedIdSet.has(company.id)
              return (
                <button
                  key={company.id}
                  type="button"
                  aria-pressed={isSelected}
                  className="flex min-h-11 w-full items-center justify-between gap-3 border-b border-slate-100 px-4 py-2 text-left text-sm last:border-b-0 hover:bg-purple-50 focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-brand-purple"
                  onClick={() => onToggle(company.id)}
                >
                  <span className="truncate">{company.name}</span>
                  <span
                    className={`flex size-6 shrink-0 items-center justify-center rounded-md border ${isSelected ? 'border-brand-purple bg-brand-purple text-white' : 'border-slate-300 text-transparent'}`}
                    aria-hidden="true"
                  >
                    <Check size={15} />
                  </span>
                </button>
              )
            })
          )}
        </div>
        {pageCount > 1 ? (
          <div className="flex items-center justify-between gap-2 border-t border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-600">
            <button
              type="button"
              className="button-ghost min-h-9 px-2"
              disabled={page <= 1 || loading}
              onClick={() => onPageChange(page - 1)}
            >
              {t('common.previous')}
            </button>
            <span>{t('users.companyPage', { page, pages: pageCount })}</span>
            <button
              type="button"
              className="button-ghost min-h-9 px-2"
              disabled={page >= pageCount || loading}
              onClick={() => onPageChange(page + 1)}
            >
              {t('common.next')}
            </button>
          </div>
        ) : null}
      </div>
    </fieldset>
  )
}
