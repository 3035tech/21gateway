import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { KeyRound, Mail } from 'lucide-react'
import { requestPasswordRecovery } from './api'
import { errorMessage } from '../../lib/admin'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from '../../components/admin/LanguageSwitcher'

export function ForgotPasswordPage() {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const request = useMutation({
    mutationFn: () => requestPasswordRecovery(email),
  })
  return (
    <main className="flex min-h-dvh items-center justify-center bg-slate-50 px-5 py-10">
      <section className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <div className="flex justify-end">
          <LanguageSwitcher />
        </div>
        <div className="flex items-center gap-3 text-brand-purple">
          <KeyRound size={28} aria-hidden="true" />
          <span className="text-xs font-bold uppercase tracking-[0.18em]">
            {t('auth.accountSecurity')}
          </span>
        </div>
        <h1 className="mt-5 text-3xl font-bold">{t('auth.recoverPassword')}</h1>
        <p className="mt-3 text-sm leading-6 text-slate-600">
          {t('auth.recoveryDescription')}
        </p>
        {request.isSuccess ? (
          <p
            role="status"
            className="mt-6 rounded-xl bg-emerald-50 p-4 text-sm text-emerald-800"
          >
            {t('auth.recoverySent')}
          </p>
        ) : (
          <form
            className="mt-6 space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              request.mutate()
            }}
          >
            <label className="field">
              {t('common.email')}
              <input
                className="form-control"
                type="email"
                autoComplete="email"
                required
                maxLength={254}
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </label>
            {request.isError ? (
              <p role="alert" className="text-sm text-rose-700">
                {errorMessage(request.error)}
              </p>
            ) : null}
            <button
              className="button-primary w-full"
              disabled={request.isPending}
            >
              <Mail size={17} aria-hidden="true" />
              {request.isPending ? t('auth.sending') : t('auth.sendLink')}
            </button>
          </form>
        )}
        <Link
          to="/login"
          className="mt-6 block text-sm font-semibold text-brand-purple hover:underline"
        >
          {t('auth.backToLogin')}
        </Link>
      </section>
    </main>
  )
}
