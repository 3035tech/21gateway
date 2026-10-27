import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { ShieldCheck } from 'lucide-react'
import { resetPassword } from './api'
import { MfaCodeInput } from './MfaCodeInput'
import { errorMessage } from '../../lib/admin'
import { systemPasswordLimits } from '../users/api'

export function ResetPasswordPage() {
  const [token] = useState(
    () => new URLSearchParams(window.location.hash.slice(1)).get('token') ?? '',
  )
  useEffect(() => {
    window.history.replaceState(
      window.history.state,
      '',
      window.location.pathname,
    )
  }, [])
  const [password, setPassword] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [factor, setFactor] = useState('')
  const [factorRequired, setFactorRequired] = useState(false)
  const [factorMode, setFactorMode] = useState<'app' | 'recovery'>('app')
  const reset = useMutation({
    mutationFn: (value?: string) =>
      resetPassword({ token, newPassword: password, factor: value ?? factor }),
    onError: (error) => {
      if (
        isAxiosError(error) &&
        ['MFA_REQUIRED', 'MFA_INVALID'].includes(error.response?.data?.code)
      )
        setFactorRequired(true)
    },
  })
  const valid = token.length === 64
  return (
    <main className="flex min-h-dvh items-center justify-center bg-slate-50 px-5 py-10">
      <section className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <div className="flex items-center gap-3 text-brand-purple">
          <ShieldCheck size={28} aria-hidden="true" />
          <span className="text-xs font-bold uppercase tracking-[0.18em]">
            Segurança da conta
          </span>
        </div>
        <h1 className="mt-5 text-3xl font-bold">Definir nova senha</h1>
        {!valid ? (
          <p role="alert" className="mt-4 text-sm text-rose-700">
            Link de recuperação inválido. Solicite outro.
          </p>
        ) : reset.isSuccess ? (
          <p
            role="status"
            className="mt-5 rounded-xl bg-emerald-50 p-4 text-sm text-emerald-800"
          >
            Senha alterada. Todas as sessões anteriores foram encerradas. Entre
            novamente.
          </p>
        ) : (
          <form
            className="mt-6 space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              if (
                password === confirmation &&
                Array.from(password.trim()).length >=
                  systemPasswordLimits.minimumCharacters
              )
                reset.mutate()
            }}
          >
            <label className="field">
              Nova senha
              <input
                className="form-control"
                type="password"
                autoComplete="new-password"
                minLength={15}
                maxLength={128}
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </label>
            <label className="field">
              Confirmar senha
              <input
                className="form-control"
                type="password"
                autoComplete="new-password"
                required
                value={confirmation}
                onChange={(event) => setConfirmation(event.target.value)}
              />
            </label>
            {password !== confirmation ? (
              <p role="alert" className="text-xs text-rose-700">
                As senhas não coincidem.
              </p>
            ) : null}
            <p className="text-xs text-slate-500">
              Use pelo menos {systemPasswordLimits.minimumCharacters} caracteres
              sem contar espaços nas pontas.
            </p>
            {factorRequired ? (
              <div className="space-y-3">
                <p className="text-sm font-semibold">Código de segurança</p>
                {factorMode === 'app' ? (
                  <MfaCodeInput
                    value={factor}
                    onChange={setFactor}
                    onComplete={(value) => {
                      if (
                        password === confirmation &&
                        Array.from(password.trim()).length >=
                          systemPasswordLimits.minimumCharacters
                      )
                        reset.mutate(value)
                    }}
                    disabled={reset.isPending}
                  />
                ) : (
                  <input
                    className="form-control w-full font-mono"
                    autoComplete="off"
                    maxLength={32}
                    placeholder="Código de recuperação"
                    value={factor}
                    onChange={(event) => setFactor(event.target.value)}
                  />
                )}
                <button
                  type="button"
                  className="button-ghost"
                  onClick={() => {
                    setFactor('')
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
            ) : null}
            {reset.isError ? (
              <p role="alert" className="text-sm text-rose-700">
                {errorMessage(reset.error)}
              </p>
            ) : null}
            <button
              className="button-primary w-full"
              disabled={
                reset.isPending ||
                password !== confirmation ||
                (factorRequired &&
                  (factorMode === 'app'
                    ? !/^\d{6}$/.test(factor)
                    : !/^[0-9a-fA-F]{32}$/.test(factor)))
              }
            >
              {reset.isPending ? 'Salvando…' : 'Salvar nova senha'}
            </button>
          </form>
        )}
        <Link
          to={valid && !reset.isSuccess ? '/esqueci-senha' : '/login'}
          className="mt-6 block text-sm font-semibold text-brand-purple hover:underline"
        >
          {valid && !reset.isSuccess
            ? 'Solicitar outro link'
            : 'Voltar ao login'}
        </Link>
      </section>
    </main>
  )
}
