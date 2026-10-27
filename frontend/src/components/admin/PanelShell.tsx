import { useRef } from 'react'
import type { PropsWithChildren } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
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
  CircleUserRound,
} from 'lucide-react'
const navigation = [
  ['/', 'Visão geral', LayoutDashboard, true],
  ['/empresas', 'Empresas', Building2, false],
  ['/usuarios', 'Usuários', Users, true],
  ['/credenciais', 'Credenciais da API', KeyRound, false],
  ['/envios', 'Envios', MessageSquare, false],
  ['/agendamentos', 'Agendamentos', CalendarClock, false],
  ['/callbacks', 'Callbacks', Webhook, false],
  ['/auditoria', 'Auditoria', History, true],
  ['/perfil', 'Meu perfil', CircleUserRound, false],
] as const
function Navigation({
  admin,
  onNavigate,
  managementOnly = false,
}: {
  admin: boolean
  onNavigate?: () => void
  managementOnly?: boolean
}) {
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
                '/perfil',
              ].includes(path)),
        )
        .map(([path, label, Icon]) => (
          <NavLink
            key={path}
            to={path}
            end={path === '/'}
            onClick={onNavigate}
            className={({ isActive }) =>
              `flex min-h-11 items-center gap-3 rounded-lg px-4 py-3 text-sm font-semibold transition-colors ${isActive ? 'bg-white text-brand-purple shadow-[inset_3px_0_0_#ff0066]' : 'text-purple-100 hover:bg-white/10'}`
            }
          >
            <Icon size={18} aria-hidden="true" />
            {label}
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
}: Props) {
  const menuRef = useRef<HTMLDetailsElement>(null)
  const location = useLocation()
  const current =
    navigation.find(([path]) => path === location.pathname)?.[1] ?? 'Painel'
  const initials = name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((value) => value[0])
    .join('')
  return (
    <div className="min-h-dvh lg:grid lg:grid-cols-[232px_minmax(0,1fr)]">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-white focus:p-4"
      >
        Ir para o conteúdo
      </a>
      <aside className="flex flex-col bg-brand-purple text-white lg:sticky lg:top-0 lg:h-dvh">
        <div className="flex items-center justify-between px-5 py-5 lg:block lg:px-7 lg:pb-8 lg:pt-8">
          <img
            src="/brand/21-mobile.png"
            alt="21 Mobile"
            className="h-10 w-auto lg:h-14"
          />
          <p className="text-xs text-purple-200 lg:mt-3">Backoffice de SMS</p>
        </div>
        <details
          ref={menuRef}
          className="group border-t border-white/15 px-3 pb-3 lg:hidden"
        >
          <summary className="flex min-h-12 cursor-pointer list-none items-center gap-3 px-3 text-sm font-semibold">
            <Menu size={18} aria-hidden="true" />
            Menu
            <span className="ml-auto font-normal text-purple-200">
              {current}
            </span>
          </summary>
          <nav aria-label="Menu principal" className="grid gap-1 pb-3">
            <Navigation
              admin={admin}
              onNavigate={() => {
                if (menuRef.current) menuRef.current.open = false
              }}
            />
          </nav>
          <div className="flex items-center justify-between border-t border-white/15 px-3 pt-3">
            <span className="text-sm">{name}</span>
            <button
              className="min-h-11 rounded px-3 text-sm"
              disabled={pending}
              onClick={onLogout}
            >
              {pending ? 'Saindo…' : 'Sair'}
            </button>
          </div>
        </details>
        <nav
          aria-label="Menu principal"
          className="hidden flex-col gap-1 px-3 lg:flex"
        >
          <Navigation admin={admin} managementOnly={managementOnly} />
        </nav>
        <div className="mt-auto hidden border-t border-white/15 p-5 lg:block">
          <div className="flex items-center gap-3">
            <span
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-white/15 text-xs font-semibold"
              aria-hidden="true"
            >
              {initials}
            </span>
            <div className="min-w-0">
              <p title={name} className="truncate text-sm font-semibold">
                {name}
              </p>
              <p className="mt-1 text-xs text-purple-200">{role}</p>
            </div>
          </div>
          <button
            disabled={pending}
            onClick={onLogout}
            className="mt-4 flex min-h-11 w-full items-center gap-2 rounded-lg px-3 text-sm text-purple-100 hover:bg-white/10"
          >
            <LogOut size={16} aria-hidden="true" />
            {pending ? 'Saindo…' : 'Sair'}
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
          {demo ? (
            <span className="rounded-md bg-purple-50 px-3 py-1 text-xs font-semibold text-brand-purple">
              Demonstração · dados fictícios
            </span>
          ) : (
            <span className="hidden text-xs text-slate-500 sm:block">
              Painel administrativo
            </span>
          )}
        </header>
        <main
          id="main-content"
          tabIndex={-1}
          className="mx-auto max-w-[1480px] px-5 py-7 outline-none sm:p-8"
        >
          {children}
          {demo ? (
            <footer className="mt-8 border-t border-slate-200 pt-5 text-xs leading-5 text-slate-500">
              Ambiente de prévia. Alterações são descartadas ao recarregar.
              Nenhum SMS é enviado.
            </footer>
          ) : null}
        </main>
      </div>
    </div>
  )
}
