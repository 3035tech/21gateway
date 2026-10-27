import type { PropsWithChildren } from 'react'
import { NavLink, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export function AppLayout({ children }: PropsWithChildren) {
  const { t } = useTranslation()
  return (
    <div className="flex min-h-dvh flex-col">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-white focus:px-4 focus:py-3 focus:text-brand-purple"
      >
        {t('nav.skipContent')}
      </a>
      <header className="border-b-[3px] border-brand-magenta bg-brand-purple text-white">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-6 px-5 py-5 sm:px-8">
          <Link
            to="/"
            aria-label={t('app.brandHome')}
            className="rounded-sm focus-visible:outline-white"
          >
            <img
              src="/brand/21-mobile.png"
              alt="21 Mobile"
              width="130"
              height="46"
            />
          </Link>
          <span className="border-l border-white/25 pl-6 text-sm font-medium text-white/90">
            {t('nav.adminPanel')}
          </span>
        </div>
      </header>
      <nav
        aria-label={t('app.mainNav')}
        className="border-b border-slate-200 bg-white"
      >
        <div className="mx-auto max-w-6xl px-5 sm:px-8">
          <NavLink
            to="/"
            end
            className={({ isActive }) =>
              `inline-flex border-b-2 px-1 py-4 text-sm font-semibold ${isActive ? 'border-brand-purple text-brand-purple' : 'border-transparent text-slate-600 hover:text-brand-purple'}`
            }
          >
            {t('nav.overview')}
          </NavLink>
        </div>
      </nav>
      <main
        id="main-content"
        tabIndex={-1}
        className="mx-auto w-full max-w-6xl flex-1 px-5 py-9 outline-none sm:px-8 sm:py-12"
      >
        {children}
      </main>
      <footer className="border-t border-slate-200">
        <div className="mx-auto flex max-w-6xl flex-wrap justify-between gap-2 px-5 py-5 text-xs text-slate-500 sm:px-8">
          <span>21 Mobile</span>
          <span>{t('app.smsPanel')}</span>
        </div>
      </footer>
    </div>
  )
}
