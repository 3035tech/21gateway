import { useQuery } from '@tanstack/react-query'
import { CircleAlert, CircleCheck, RefreshCw, Radio, Users } from 'lucide-react'
import { getReadiness } from '../../lib/api'

const timeFormatter = new Intl.DateTimeFormat('pt-BR', {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
})

export function SystemPage() {
  const health = useQuery({
    queryKey: ['system', 'readiness'],
    queryFn: ({ signal }) => getReadiness(signal),
  })
  const offline = health.fetchStatus === 'paused'
  const checking = !offline && (health.isFetching || health.isPending)
  const unavailable = offline || health.isError
  const checkedAt = Math.max(health.dataUpdatedAt, health.errorUpdatedAt)
  const StatusIcon = checking
    ? RefreshCw
    : unavailable
      ? CircleAlert
      : CircleCheck
  const statusTitle = offline
    ? 'Sem conexão com a internet'
    : checking
      ? 'Verificando conexão'
      : health.isError
        ? 'Conexão indisponível'
        : 'Conectado à API'
  const statusDescription = offline
    ? 'Restabeleça sua conexão para verificar a disponibilidade do serviço.'
    : checking
      ? 'Aguarde enquanto verificamos a disponibilidade do serviço.'
      : health.isError
        ? 'Não foi possível acessar o serviço. Tente novamente em alguns instantes.'
        : 'O serviço respondeu à última verificação de conexão.'

  return (
    <>
      <div className="mb-8 max-w-2xl">
        <h1 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl">
          Visão geral
        </h1>
        <p className="mt-3 leading-7 text-slate-600">
          Confira a conexão com o serviço de SMS e acompanhe a preparação do seu
          painel.
        </p>
      </div>
      <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <section
          className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm"
          aria-labelledby="connection-title"
        >
          <div className="flex items-center gap-3 border-b border-slate-100 px-6 py-5 sm:px-7">
            <Radio size={20} className="text-brand-purple" aria-hidden="true" />
            <h2 id="connection-title" className="text-base font-bold">
              Conexão com a API
            </h2>
          </div>
          <div className="px-6 py-7 sm:px-7">
            <div
              role="status"
              aria-live="polite"
              aria-atomic="true"
              className="flex items-start gap-4"
            >
              <div
                className={`flex size-11 shrink-0 items-center justify-center rounded-full ${checking ? 'bg-purple-50 text-brand-purple' : unavailable ? 'bg-amber-50 text-amber-700' : 'bg-emerald-50 text-emerald-700'}`}
              >
                <StatusIcon
                  size={22}
                  className={checking ? 'motion-safe:animate-spin' : undefined}
                  aria-hidden="true"
                />
              </div>
              <div>
                <h3 className="font-semibold text-slate-900">{statusTitle}</h3>
                <p className="mt-2 max-w-md text-sm leading-6 text-slate-600">
                  {statusDescription}
                </p>
              </div>
            </div>
            <div className="mt-7 flex flex-wrap items-center justify-between gap-4 border-t border-slate-100 pt-5">
              <button
                type="button"
                disabled={checking || health.isFetching}
                onClick={() => void health.refetch()}
                className="button-primary"
              >
                <RefreshCw size={16} aria-hidden="true" />
                {checking ? 'Verificando…' : 'Verificar novamente'}
              </button>
              <span className="text-xs text-slate-500">
                {checkedAt ? (
                  <>
                    Última verificação{' '}
                    <time dateTime={new Date(checkedAt).toISOString()}>
                      {timeFormatter.format(checkedAt)}
                    </time>
                  </>
                ) : (
                  'A primeira verificação está em andamento.'
                )}
              </span>
            </div>
          </div>
        </section>
        <aside
          className="rounded-xl border border-purple-100 bg-purple-50/60 p-6"
          aria-labelledby="management-title"
        >
          <Users size={23} className="text-brand-purple" aria-hidden="true" />
          <h2
            id="management-title"
            className="mt-4 text-base font-bold text-brand-purple"
          >
            Gestão de usuários e empresas
          </h2>
          <p className="mt-3 text-sm leading-6 text-slate-600">
            Os cadastros e as permissões de acesso serão disponibilizados neste
            painel na próxima etapa.
          </p>
          <p className="mt-5 border-t border-purple-100 pt-4 text-xs leading-5 text-slate-500">
            A verificação de conexão não confirma o envio nem a entrega de
            mensagens.
          </p>
        </aside>
      </div>
    </>
  )
}
