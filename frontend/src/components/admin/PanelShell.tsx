import { useRef, useState } from 'react'
import type { PropsWithChildren } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from './LanguageSwitcher'
import { AppSelect } from './AppSelect'
import { NotificationBell } from '../../features/notifications/NotificationBell'
import {
  Building2,
  Users,
  KeyRound,
  MessageSquare,
  CalendarClock,
  Webhook,
  History,
  LayoutDashboard,
  LogOut,
  Menu,
  ChevronRight,
  Bell,
  ChevronLeft,
  CircleUserRound,
} from 'lucide-react'
const sidebarPreferenceKey = '21mobile.sidebar.collapsed'

function readSidebarPreference() {
  try {
    return window.localStorage.getItem(sidebarPreferenceKey) === 'true'
  } catch {
    return false
  }
}
const navigation = [
  ['/', 'nav.overview', LayoutDashboard, false],
  ['/empresas', 'nav.companies', Building2, false],
  ['/usuarios', 'nav.users', Users, true],
  ['/credenciais', 'nav.credentials', KeyRound, false],
  ['/envios', 'nav.messages', MessageSquare, false],
  ['/agendamentos', 'nav.schedules', CalendarClock, false],
  ['/callbacks', 'nav.callbacks', Webhook, false],
  ['/auditoria', 'nav.audit', History, true],
  ['/notificacoes', 'nav.notifications', Bell, true],
] as const
function Navigation({
  admin,
  onNavigate,
  managementOnly = false,
  collapsed = false,
}: {
  admin: boolean
  onNavigate?: () => void
  managementOnly?: boolean
  collapsed?: boolean
}) {
  const { t } = useTranslation()
  return (
    <>
      {navigation
        .filter(
          ([path, , , restricted]) =>
            (!restricted || admin) &&
            (!managementOnly ||
              [
                '/',
                '/empresas',
                '/usuarios',
                '/credenciais',
                '/agendamentos',
              ].includes(path)),
        )
        .map(([path, label, Icon]) => (
          <NavLink
            key={path}
            to={path}
            end={path === '/'}
            onClick={onNavigate}
            title={collapsed ? t(label) : undefined}
            aria-label={collapsed ? t(label) : undefined}
            className={({ isActive }) =>
              `flex min-h-11 items-center gap-3 rounded-lg py-3 text-sm font-semibold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white ${collapsed ? 'justify-center px-2' : 'px-4'} ${isActive ? 'bg-white text-brand-purple shadow-[inset_3px_0_0_#ff0066]' : 'text-purple-100 hover:bg-white/10'}`
            }
          >
            <Icon size={18} aria-hidden="true" />
            {collapsed ? <span className="sr-only">{t(label)}</span> : t(label)}
          </NavLink>
        ))}
    </>
  )
}
type Props = PropsWithChildren<{
  name: string
  role: string
  admin: boolean
  managementOnly?: boolean
  demo?: boolean
  pending?: boolean
  logoutError?: string
  onLogout: () => void
  companyView?: {
    value: string | number
    options: { value: string | number; label: string }[]
    onChange: (value: string) => void
  }
}>
export function PanelShell({
  children,
  name,
  role,
  admin,
  demo = false,
  managementOnly = false,
  pending = false,
  logoutError,
  onLogout,
  companyView,
}: Props) {
  const { t } = useTranslation()
  const menuRef = useRef<HTMLDetailsElement>(null)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(
    readSidebarPreference,
  )
  const toggleSidebar = () => {
    const nextValue = !sidebarCollapsed
    setSidebarCollapsed(nextValue)
    try {
      window.localStorage.setItem(sidebarPreferenceKey, String(nextValue))
    } catch {
      // Keep the current session usable when storage is unavailable.
    }
  }
  const location = useLocation()
  const currentKey =
    location.pathname === '/perfil'
      ? 'nav.profile'
      : (navigation.find(([path]) => path === location.pathname)?.[1] ??
        'nav.adminPanel')
  const current = t(currentKey)
  const initials = name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((value) => value[0])
    .join('')
  return (
    <div
      className={`min-h-dvh overflow-x-clip lg:grid ${sidebarCollapsed ? 'lg:grid-cols-[76px_minmax(0,1fr)]' : 'lg:grid-cols-[232px_minmax(0,1fr)]'}`}
    >
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-white focus:p-4"
      >
        {t('nav.skipContent')}
      </a>
      <aside className="relative flex flex-col bg-brand-purple text-white lg:sticky lg:top-0 lg:h-dvh">
        <div
          className={`flex items-center justify-between px-5 py-5 lg:block lg:pb-8 lg:pt-8 ${sidebarCollapsed ? 'lg:px-3' : 'lg:px-7'}`}
        >
          <div className="flex items-center gap-3 lg:block">
            <img
              src="/brand/21-mobile.png"
              alt="21 Mobile"
              className={`h-10 w-auto lg:h-14 ${sidebarCollapsed ? 'lg:hidden' : ''}`}
            />
            {sidebarCollapsed ? (
              <span
                className="hidden h-14 items-center justify-center text-2xl font-semibold tracking-tighter lg:flex"
                aria-label="21 Mobile"
              >
                21
              </span>
            ) : null}
            <p
              className={`text-xs text-purple-200 lg:mt-3 ${sidebarCollapsed ? 'lg:hidden' : ''}`}
            >
              {t('nav.backoffice')}
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={toggleSidebar}
          className="group absolute -right-[22px] top-8 z-30 hidden size-11 items-center justify-center rounded-full focus-visible:outline-none lg:flex"
          aria-label={t(
            sidebarCollapsed ? 'nav.expandSidebar' : 'nav.collapseSidebar',
          )}
          title={t(
            sidebarCollapsed ? 'nav.expandSidebar' : 'nav.collapseSidebar',
          )}
          aria-expanded={!sidebarCollapsed}
        >
          <span className="flex size-8 items-center justify-center rounded-full border border-purple-200 bg-white text-brand-purple shadow-sm transition-colors group-hover:border-brand-purple group-hover:bg-purple-50 group-focus-visible:ring-2 group-focus-visible:ring-brand-magenta group-focus-visible:ring-offset-2">
            <ChevronLeft
              size={17}
              strokeWidth={2.3}
              className={sidebarCollapsed ? 'rotate-180' : ''}
              aria-hidden="true"
            />
          </span>
        </button>
        <details
          ref={menuRef}
          className="group border-t border-white/15 px-3 pb-3 lg:hidden"
        >
          <summary className="flex min-h-12 cursor-pointer list-none items-center gap-3 rounded-lg px-3 text-sm font-semibold transition-colors hover:bg-white/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white">
            <Menu size={18} aria-hidden="true" />
            {t('nav.menu')}
            <span className="ml-auto font-normal text-purple-200">
              {current}
            </span>
          </summary>
          <nav aria-label={t('nav.menu')} className="grid gap-1 pb-3">
            <Navigation
              admin={admin}
              onNavigate={() => {
                if (menuRef.current) menuRef.current.open = false
              }}
            />
          </nav>
          <div className="flex items-center justify-between border-t border-white/15 px-3 pt-3">
            <NavLink
              to="/perfil"
              onClick={() => {
                if (menuRef.current) menuRef.current.open = false
              }}
              className="min-h-11 min-w-0 rounded-lg px-2 py-3 text-sm font-semibold hover:bg-white/10 focus-visible:outline-2 focus-visible:outline-white"
              aria-label={t('nav.openProfile', { name })}
            >
              {name}
            </NavLink>
            <button
              className="button-on-dark"
              disabled={pending}
              onClick={onLogout}
            >
              {pending ? t('nav.loggingOut') : t('nav.logout')}
            </button>
          </div>
        </details>
        <nav
          aria-label={t('nav.menu')}
          className="hidden flex-col gap-1 px-3 lg:flex"
        >
          <Navigation
            admin={admin}
            managementOnly={managementOnly}
            collapsed={sidebarCollapsed}
          />
        </nav>
        <div
          className={`mt-auto hidden border-t border-white/15 lg:block ${sidebarCollapsed ? 'p-3' : 'p-5'}`}
        >
          <NavLink
            to="/perfil"
            aria-label={t('nav.openProfile', { name })}
            title={
              sidebarCollapsed ? t('nav.openProfile', { name }) : undefined
            }
            className={({ isActive }) =>
              `flex min-h-14 items-center gap-3 rounded-lg p-2 transition-colors hover:bg-white/10 focus-visible:outline-2 focus-visible:outline-white ${sidebarCollapsed ? 'justify-center' : ''} ${isActive ? 'bg-white/15' : ''}`
            }
          >
            {sidebarCollapsed ? (
              <CircleUserRound size={24} aria-hidden="true" />
            ) : (
              <span
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-white/15 text-xs font-semibold"
                aria-hidden="true"
              >
                {initials}
              </span>
            )}
            {!sidebarCollapsed ? (
              <div className="min-w-0">
                <p title={name} className="truncate text-sm font-semibold">
                  {name}
                </p>
                <p className="mt-1 text-xs text-purple-200">{role}</p>
              </div>
            ) : null}
          </NavLink>
          <button
            disabled={pending}
            onClick={onLogout}
            className={`button-on-dark mt-4 w-full ${sidebarCollapsed ? 'justify-center px-2' : ''}`}
            aria-label={
              sidebarCollapsed
                ? pending
                  ? t('nav.loggingOut')
                  : t('nav.logout')
                : undefined
            }
            title={
              sidebarCollapsed
                ? pending
                  ? t('nav.loggingOut')
                  : t('nav.logout')
                : undefined
            }
          >
            <LogOut size={16} aria-hidden="true" />
            {!sidebarCollapsed
              ? pending
                ? t('nav.loggingOut')
                : t('nav.logout')
              : null}
          </button>
        </div>
        {logoutError ? (
          <p role="alert" className="px-5 pb-4 text-sm text-rose-100">
            {logoutError}
          </p>
        ) : null}
      </aside>
      <div className="min-w-0">
        <header className="flex min-h-16 flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-white px-5 sm:px-8">
          <div className="flex items-center gap-2 text-sm">
            <span className="text-slate-500">21 Mobile</span>
            <ChevronRight
              size={14}
              aria-hidden="true"
              className="text-slate-400"
            />
            <span className="font-semibold text-slate-700">{current}</span>
          </div>
          <div className="flex items-center gap-3">
            {companyView ? (
              <div className="min-w-40 max-w-64">
                <AppSelect
                  value={companyView.value}
                  label={t('common.viewingCompany')}
                  onValueChange={companyView.onChange}
                  options={companyView.options}
                />
              </div>
            ) : null}
            {admin && !demo ? <NotificationBell /> : null}
            <LanguageSwitcher />
            {demo ? (
              <span className="hidden rounded-md bg-purple-50 px-3 py-1 text-xs font-semibold text-brand-purple sm:block">
                {t('nav.demoBadge')}
              </span>
            ) : (
              <span className="hidden text-xs text-slate-500 lg:block">
                {t('nav.adminPanel')}
              </span>
            )}
          </div>
        </header>
        <main
          id="main-content"
          tabIndex={-1}
          className={`mx-auto px-5 py-7 outline-none sm:p-8 ${sidebarCollapsed ? 'max-w-[1600px]' : 'max-w-[1480px]'}`}
        >
          {children}
          {demo ? (
            <footer className="mt-8 border-t border-slate-200 pt-5 text-xs leading-5 text-slate-500">
              {t('nav.demoFootnote')}
            </footer>
          ) : null}
        </main>
      </div>
    </div>
  )
}
