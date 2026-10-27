import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'
import { ShieldCheck, ShieldOff, Copy, KeyRound } from 'lucide-react'
import { beginMFA, confirmMFA, disableMFA, mfaStatus } from '../auth/api'
import { MfaCodeInput } from '../auth/MfaCodeInput'
import { errorMessage } from '../../lib/admin'

export function MFASettings() {
  const client = useQueryClient()
  const status = useQuery({ queryKey: ['admin', 'mfa'], queryFn: mfaStatus })
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [factorMode, setFactorMode] = useState<'app' | 'recovery'>('app')
  const [setup, setSetup] = useState<{ secret: string; uri: string } | null>(
    null,
  )
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
  const begin = useMutation({
    mutationFn: () => beginMFA(password),
    onSuccess: setSetup,
  })
  const confirm = useMutation({
    mutationFn: (value: string) => confirmMFA(value),
    onSuccess: (value) => {
      setRecoveryCodes(value.recoveryCodes)
      setSetup(null)
      setPassword('')
      setCode('')
      client.setQueryData(['admin', 'mfa'], { enabled: true })
    },
  })
  const disable = useMutation({
    mutationFn: (value: string) => disableMFA(password, value),
    onSuccess: () => {
      setPassword('')
      setCode('')
      client.setQueryData(['admin', 'mfa'], { enabled: false })
    },
  })
  return (
    <section
      className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
      aria-labelledby="mfa-heading"
    >
      <h2
        id="mfa-heading"
        className="flex items-center gap-2 text-lg font-bold"
      >
        <ShieldCheck size={20} aria-hidden="true" /> Verificação em duas etapas
      </h2>
      <p className="mt-2 text-sm text-slate-500">
        Use um aplicativo autenticador para proteger o acesso ao painel. Guarde
        os códigos de recuperação em local seguro.
      </p>
      {status.isPending ? (
        <p role="status" className="mt-5 text-sm text-slate-500">
          Carregando estado de segurança…
        </p>
      ) : null}
      {status.isError ? (
        <p role="alert" className="mt-5 text-sm text-rose-700">
          {errorMessage(status.error)}
        </p>
      ) : null}
      {status.data ? (
        <div className="mt-5">
          <span
            className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-bold ${status.data.enabled ? 'bg-emerald-50 text-emerald-800' : 'bg-amber-50 text-amber-800'}`}
          >
            {status.data.enabled ? 'Ativada' : 'Desativada'}
          </span>
          {recoveryCodes.length > 0 ? (
            <div className="mt-5 rounded-xl border border-emerald-200 bg-emerald-50 p-5">
              <h3 className="font-bold text-emerald-950">
                Códigos de recuperação
              </h3>
              <p className="mt-2 text-sm text-emerald-900">
                Cada código funciona uma única vez. Eles aparecem somente agora.
                Copie e guarde fora deste computador.
              </p>
              <div className="mt-4 grid gap-2 font-mono text-sm sm:grid-cols-2">
                {recoveryCodes.map((value) => (
                  <code key={value}>{value}</code>
                ))}
              </div>
              <button
                type="button"
                className="mt-4 inline-flex min-h-11 items-center gap-2 rounded-lg border border-emerald-300 bg-white px-4 text-sm font-semibold text-emerald-900 transition-colors hover:bg-emerald-100 active:bg-emerald-200"
                onClick={() =>
                  void navigator.clipboard.writeText(recoveryCodes.join('\n'))
                }
              >
                <Copy size={16} aria-hidden="true" /> Copiar códigos
              </button>
              <button
                type="button"
                className="ml-3 min-h-11 rounded-lg px-3 text-sm font-semibold text-emerald-900 underline transition-colors hover:bg-emerald-100 active:bg-emerald-200"
                onClick={() => setRecoveryCodes([])}
              >
                Já salvei os códigos
              </button>
            </div>
          ) : null}
          {!status.data.enabled && !setup ? (
            <form
              className="mt-5 max-w-md space-y-3"
              onSubmit={(event) => {
                event.preventDefault()
                begin.mutate()
              }}
            >
              <label className="field">
                Confirme sua senha
                <input
                  className="form-control"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </label>
              {begin.isError ? (
                <p role="alert" className="text-sm text-rose-700">
                  {errorMessage(begin.error)}
                </p>
              ) : null}
              <button
                type="submit"
                disabled={begin.isPending}
                className="button-primary"
              >
                <KeyRound size={16} aria-hidden="true" /> Configurar aplicativo
              </button>
            </form>
          ) : null}
          {setup ? (
            <div className="mt-5 space-y-4">
              <p className="text-sm font-medium">
                Escaneie o QR code no aplicativo autenticador. Se preferir,
                digite a chave manualmente.
              </p>
              <div className="w-fit rounded-xl border border-slate-200 bg-white p-4">
                <QRCodeSVG
                  value={setup.uri}
                  size={192}
                  marginSize={4}
                  title="QR code para configurar a autenticação em duas etapas"
                />
              </div>
              <div>
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Chave manual
                </p>
                <code className="mt-1 block break-all text-sm font-semibold">
                  {setup.secret}
                </code>
              </div>
              <p className="text-xs text-slate-500">
                Este QR code contém uma chave secreta. Não compartilhe a tela. A
                configuração expira em dez minutos.
              </p>
              <form
                className="max-w-md space-y-3"
                onSubmit={(event) => {
                  event.preventDefault()
                  if (/^\d{6}$/.test(code)) confirm.mutate(code)
                }}
              >
                <p className="text-sm font-semibold">Código de 6 dígitos</p>
                <MfaCodeInput
                  value={code}
                  onChange={setCode}
                  onComplete={(value) => confirm.mutate(value)}
                  disabled={confirm.isPending}
                />
                {confirm.isError ? (
                  <p role="alert" className="text-sm text-rose-700">
                    {errorMessage(confirm.error)}
                  </p>
                ) : null}
                <button
                  className="button-primary"
                  disabled={confirm.isPending || !/^\d{6}$/.test(code)}
                >
                  Ativar verificação
                </button>
              </form>
            </div>
          ) : null}
          {status.data.enabled && recoveryCodes.length === 0 ? (
            <form
              className="mt-5 max-w-md space-y-3"
              onSubmit={(event) => {
                event.preventDefault()
                if (
                  password &&
                  (factorMode === 'app'
                    ? /^\d{6}$/.test(code)
                    : /^[0-9a-fA-F]{32}$/.test(code))
                )
                  disable.mutate(code)
              }}
            >
              <h3 className="flex items-center gap-2 text-sm font-bold">
                <ShieldOff size={17} aria-hidden="true" /> Desativar segundo
                fator
              </h3>
              <p className="text-xs text-slate-500">
                Exige sua senha e um código válido. Todos os códigos de
                recuperação serão invalidados.
              </p>
              <label className="field">
                Senha atual
                <input
                  className="form-control"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </label>
              <div className="space-y-3">
                <p className="text-sm font-semibold">Código de segurança</p>
                {factorMode === 'app' ? (
                  <MfaCodeInput
                    value={code}
                    onChange={setCode}
                    onComplete={(value) => {
                      if (password) disable.mutate(value)
                    }}
                    disabled={disable.isPending}
                  />
                ) : (
                  <input
                    className="form-control w-full font-mono"
                    autoComplete="off"
                    maxLength={32}
                    placeholder="Código de recuperação"
                    value={code}
                    onChange={(event) => setCode(event.target.value)}
                  />
                )}
                <button
                  type="button"
                  className="button-ghost"
                  onClick={() => {
                    setCode('')
                    setFactorMode((mode) =>
                      mode === 'app' ? 'recovery' : 'app',
                    )
                  }}
                >
                  {factorMode === 'app'
                    ? 'Usar código de recuperação'
                    : 'Usar aplicativo autenticador'}
                </button>
              </div>
              {disable.isError ? (
                <p role="alert" className="text-sm text-rose-700">
                  {errorMessage(disable.error)}
                </p>
              ) : null}
              <button
                className="button-danger"
                disabled={
                  disable.isPending ||
                  !password ||
                  (factorMode === 'app'
                    ? !/^\d{6}$/.test(code)
                    : !/^[0-9a-fA-F]{32}$/.test(code))
                }
              >
                Desativar verificação
              </button>
            </form>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
