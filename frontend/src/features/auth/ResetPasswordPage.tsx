import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { ShieldCheck } from 'lucide-react'
import { resetPassword } from './api'
import { MfaCodeInput } from './MfaCodeInput'
import { errorMessage } from '../../lib/admin'
import { systemPasswordLimits } from '../users/api'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from '../../components/admin/LanguageSwitcher'

export function ResetPasswordPage() {
  const { t } = useTranslation()
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
        <div className="flex justify-end">
          <LanguageSwitcher />
        </div>
        <div className="flex items-center gap-3 text-brand-purple">
          <ShieldCheck size={28} aria-hidden="true" />
          <span className="text-xs font-bold uppercase tracking-[0.18em]">
            {t('auth.accountSecurity')}
          </span>
        </div>
        <h1 className="mt-5 text-3xl font-bold">{t('auth.setNewPassword')}</h1>
        {!valid ? (
          <p role="alert" className="mt-4 text-sm text-rose-700">
            {t('errors.INVALID_RESET_TOKEN')}
          </p>
        ) : reset.isSuccess ? (
          <p
            role="status"
            className="mt-5 rounded-xl bg-emerald-50 p-4 text-sm text-emerald-800"
          >
            {t('auth.passwordChanged')}
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
              {t('auth.newPassword')}
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
              {t('auth.confirmPassword')}
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
                {t('validation.passwordMismatch')}
              </p>
            ) : null}
            <p className="text-xs text-slate-500">
              {t('auth.passwordHint', {
                count: systemPasswordLimits.minimumCharacters,
              })}
            </p>
            {factorRequired ? (
              <div className="space-y-3">
                <p className="text-sm font-semibold">
                  {t('auth.securityCode')}
                </p>
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
                    placeholder={t('auth.recoveryCode')}
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
                    ? t('auth.useRecoveryCode')
                    : t('auth.useAuthenticator')}
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
              {reset.isPending ? t('common.saving') : t('auth.saveNewPassword')}
            </button>
          </form>
        )}
        <Link
          to={valid && !reset.isSuccess ? '/esqueci-senha' : '/login'}
          className="mt-6 block text-sm font-semibold text-brand-purple hover:underline"
        >
          {valid && !reset.isSuccess
            ? t('auth.requestAnotherLink')
            : t('auth.backToLogin')}
        </Link>
      </section>
    </main>
  )
}
