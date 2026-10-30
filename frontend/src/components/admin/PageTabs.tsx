import type { ReactNode, KeyboardEvent } from 'react'
import type { LucideIcon } from 'lucide-react'

export type PageTab<T extends string> = {
  id: T
  label: string
  icon: LucideIcon
}

export function PageTabs<T extends string>({
  idPrefix,
  label,
  tabs,
  active,
  onChange,
}: {
  idPrefix: string
  label: string
  tabs: readonly PageTab<T>[]
  active: T
  onChange: (tab: T) => void
}) {
  const onKeyDown = (
    event: KeyboardEvent<HTMLButtonElement>,
    index: number,
  ) => {
    const next =
      event.key === 'ArrowRight'
        ? (index + 1) % tabs.length
        : event.key === 'ArrowLeft'
          ? (index - 1 + tabs.length) % tabs.length
          : event.key === 'Home'
            ? 0
            : event.key === 'End'
              ? tabs.length - 1
              : null
    if (next === null) return
    event.preventDefault()
    onChange(tabs[next].id)
    const buttons =
      event.currentTarget.parentElement?.querySelectorAll<HTMLButtonElement>(
        '[role="tab"]',
      )
    buttons?.[next]?.focus()
  }

  return (
    <div
      role="tablist"
      aria-label={label}
      className="flex gap-1 overflow-x-auto border-b border-slate-200"
    >
      {tabs.map(({ id, label: tabLabel, icon: Icon }, index) => (
        <button
          key={id}
          id={`${idPrefix}-tab-${id}`}
          type="button"
          role="tab"
          aria-selected={active === id}
          aria-controls={`${idPrefix}-panel-${id}`}
          tabIndex={active === id ? 0 : -1}
          onClick={() => onChange(id)}
          onKeyDown={(event) => onKeyDown(event, index)}
          className={`inline-flex min-h-12 shrink-0 items-center gap-2 border-b-2 px-4 py-3 text-sm font-semibold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-purple ${active === id ? 'border-brand-purple text-brand-purple' : 'border-transparent text-slate-600 hover:border-slate-300 hover:text-slate-900'}`}
        >
          <Icon size={17} aria-hidden="true" />
          {tabLabel}
        </button>
      ))}
    </div>
  )
}

export function PageTabPanel({
  idPrefix,
  tab,
  active,
  children,
}: {
  idPrefix: string
  tab: string
  active: boolean
  children: ReactNode
}) {
  return (
    <div
      id={`${idPrefix}-panel-${tab}`}
      role="tabpanel"
      aria-labelledby={`${idPrefix}-tab-${tab}`}
      hidden={!active}
      tabIndex={0}
      className="space-y-6 outline-none"
    >
      {children}
    </div>
  )
}
