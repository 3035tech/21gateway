import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import {
  ArrowUpRight,
  Building2,
  CalendarClock,
  KeyRound,
  RefreshCw,
  Send,
  Webhook,
} from 'lucide-react'
import { getSummary } from './api'
import { errorMessage } from '../../lib/admin'

const number = (value: number) => value.toLocaleString('pt-BR')

export function OverviewPage() {
  const query = useQuery({
    queryKey: ['admin', 'summary'],
    queryFn: ({ signal }) => getSummary(signal),
  })
  const data = query.data
  return (
    <>
      <header className="mb-7 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-950 sm:text-3xl">
            Visão geral
          </h1>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            Acompanhe a fila e encontre o que precisa de atenção.
          </p>
        </div>
        <button
          className="button-secondary"
          disabled={query.isFetching}
          onClick={() => void query.refetch()}
        >
          <RefreshCw size={16} aria-hidden="true" /> Atualizar dados
        </button>
      </header>
      {query.isPending ? (
        <p role="status" className="rounded-xl bg-white p-8">
          Carregando indicadores…
        </p>
      ) : query.isError ? (
        <p role="alert" className="rounded-xl bg-rose-50 p-6 text-rose-800">
          {errorMessage(query.error)}
        </p>
      ) : data ? (
        <div className="space-y-6">
          <section className="overflow-hidden rounded-2xl bg-brand-purple text-white">
            <div className="grid lg:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]">
              <div className="p-6 sm:p-8">
                <h2 className="flex items-center gap-2 text-sm font-semibold text-purple-100">
                  <CalendarClock size={19} aria-hidden="true" /> Fila de
                  agendamentos
                </h2>
                <div className="mt-5 flex flex-wrap items-end gap-x-5 gap-y-2">
                  <strong className="text-6xl font-bold leading-none tabular-nums tracking-tight sm:text-7xl">
                    {number(data.scheduledDue)}
                  </strong>
                  <p className="max-w-56 pb-1 text-sm leading-6 text-purple-100">
                    aguardando despacho após o horário previsto
                  </p>
                </div>
                <Link
                  to="/agendamentos"
                  className="mt-7 inline-flex min-h-11 items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-bold text-brand-purple hover:bg-purple-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
                >
                  Ver agendamentos <ArrowUpRight size={17} aria-hidden="true" />
                </Link>
              </div>
              <div className="grid grid-cols-3 border-t border-white/20 bg-white/10 lg:grid-cols-1 lg:border-l lg:border-t-0">
                {[
                  ['Próximas 24 h', data.scheduledNext24h],
                  ['Em processamento', data.scheduledActive],
                  ['Total na fila', data.scheduledQueued],
                ].map(([label, value]) => (
                  <div
                    key={label}
                    className="flex flex-col justify-center border-r border-white/15 p-4 last:border-r-0 lg:border-b lg:border-r-0 lg:px-7 lg:py-5 lg:last:border-b-0"
                  >
                    <span className="text-xs leading-5 text-purple-100 sm:text-sm">
                      {label}
                    </span>
                    <strong className="mt-1 text-2xl font-bold tabular-nums sm:text-3xl">
                      {number(Number(value))}
                    </strong>
                  </div>
                ))}
              </div>
            </div>
          </section>

          <div className="grid gap-5 lg:grid-cols-2">
            <section className="rounded-2xl border border-slate-200 bg-white p-6 sm:p-7">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="flex items-center gap-2 text-lg font-bold text-slate-950">
                    <Send
                      size={20}
                      className="text-brand-purple"
                      aria-hidden="true"
                    />{' '}
                    Envios registrados
                  </h2>
                  <p className="mt-1 text-xs text-slate-500">
                    Acumulado do histórico mantido no banco
                  </p>
                </div>
                <Link
                  to="/envios"
                  className="inline-flex min-h-11 items-center gap-1 text-sm font-bold text-brand-purple hover:underline"
                >
                  Abrir <ArrowUpRight size={16} aria-hidden="true" />
                </Link>
              </div>
              <dl className="mt-6 grid grid-cols-3 gap-3 border-t border-slate-100 pt-6">
                {[
                  ['Aceitos', data.accepted],
                  ['Rejeitados', data.rejected],
                  ['Incerto', data.unknown],
                ].map(([label, value]) => (
                  <div key={label}>
                    <dt className="text-xs text-slate-500 sm:text-sm">
                      {label}
                    </dt>
                    <dd className="mt-1 text-2xl font-bold tabular-nums text-slate-900">
                      {number(Number(value))}
                    </dd>
                  </div>
                ))}
              </dl>
              <p className="mt-6 rounded-lg bg-amber-50 px-4 py-3 text-xs leading-5 text-amber-900">
                Aceite pelo fornecedor não confirma entrega. Resultado incerto
                exige reconciliação antes de qualquer reenvio.
              </p>
            </section>

            <section className="rounded-2xl border border-slate-200 bg-white p-6 sm:p-7">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="flex items-center gap-2 text-lg font-bold text-slate-950">
                    <Webhook
                      size={20}
                      className="text-brand-purple"
                      aria-hidden="true"
                    />{' '}
                    Notificações aos clientes
                  </h2>
                  <p className="mt-1 text-xs text-slate-500">
                    Jobs pendentes e falhos de DLR e MO
                  </p>
                </div>
                <Link
                  to="/callbacks"
                  className="inline-flex min-h-11 items-center gap-1 text-sm font-bold text-brand-purple hover:underline"
                >
                  Ver DLR <ArrowUpRight size={16} aria-hidden="true" />
                </Link>
              </div>
              <div className="mt-6 grid grid-cols-2 gap-4 border-t border-slate-100 pt-6">
                {[
                  ['Status DLR', data.pendingCallbacks, data.failedCallbacks],
                  [
                    'Respostas MO',
                    data.pendingMoCallbacks,
                    data.failedMoCallbacks,
                  ],
                ].map(([label, pending, failed]) => (
                  <div key={label}>
                    <h3 className="text-sm font-bold text-slate-700">
                      {label}
                    </h3>
                    <p className="mt-2 text-2xl font-bold tabular-nums text-slate-900">
                      {number(Number(pending))}{' '}
                      <span className="text-xs font-normal text-slate-500">
                        pendentes
                      </span>
                    </p>
                    <p className="mt-1 text-sm text-rose-700">
                      {number(Number(failed))} com falha
                    </p>
                  </div>
                ))}
              </div>
            </section>
          </div>

          <section className="flex flex-wrap items-center gap-x-6 gap-y-3 rounded-2xl border border-slate-200 bg-white px-6 py-5 text-sm">
            <span className="font-bold text-slate-900">Acessos ativos</span>
            <span className="inline-flex items-center gap-2 text-slate-700">
              <Building2
                size={17}
                className="text-brand-purple"
                aria-hidden="true"
              />{' '}
              {number(data.companies)} empresas
            </span>
            <span className="inline-flex items-center gap-2 text-slate-700">
              <KeyRound
                size={17}
                className="text-brand-purple"
                aria-hidden="true"
              />{' '}
              {number(data.activeCredentials)} credenciais
            </span>
            <Link
              to="/credenciais"
              className="ml-auto inline-flex min-h-11 items-center gap-1 font-bold text-brand-purple hover:underline"
            >
              Gerenciar acessos <ArrowUpRight size={16} aria-hidden="true" />
            </Link>
          </section>
        </div>
      ) : null}
    </>
  )
}
