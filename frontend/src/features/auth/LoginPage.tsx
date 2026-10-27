import { useId, useState } from 'react'
import { Eye, EyeOff, LoaderCircle } from 'lucide-react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, Navigate } from 'react-router-dom'
import { isAxiosError } from 'axios'
import { login } from './api'
import { MfaCodeInput } from './MfaCodeInput'
import { useUser } from './context'
import { errorMessage } from '../../lib/admin'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from '../../components/admin/LanguageSwitcher'
const schema = z.object({
  email: z.email(),
  password: z.string().min(1).max(128),
  factor: z.string().max(32).optional(),
  remember: z.boolean().optional(),
})
export function LoginPage() {
  const { t } = useTranslation()
  const id = useId()
  const [visible, setVisible] = useState(false)
  const [factorRequired, setFactorRequired] = useState(false)
  const [factorMode, setFactorMode] = useState<'app' | 'recovery'>('app')
  const user = useUser()
  const client = useQueryClient()
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  })
  const factor = useWatch({ control: form.control, name: 'factor' }) ?? ''
  const mutation = useMutation({
    mutationFn: login,
    onSuccess: async (value) => {
      client.removeQueries({ queryKey: ['admin'] })
      client.setQueryData(['session'], value)
      form.reset()
    },
    onError: (error) => {
      if (
        isAxiosError(error) &&
        ['MFA_REQUIRED', 'MFA_INVALID'].includes(error.response?.data?.code)
      )
        setFactorRequired(true)
    },
  })
  if (user) return <Navigate to="/" replace />
  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      <div className="hidden flex-col justify-between bg-brand-purple p-14 text-white lg:flex">
        <img src="/brand/21-mobile.png" alt="21 Mobile" className="w-48" />
        <div>
          <h1 className="max-w-md text-4xl font-semibold leading-tight">
            {t('auth.heroTitle')}
          </h1>
          <p className="mt-5 max-w-sm text-base leading-7 text-purple-100">
            {t('auth.heroDescription')}
          </p>
        </div>
        <p className="text-sm text-purple-200">{t('auth.brandFooter')}</p>
      </div>
      <main className="flex items-center justify-center px-6 py-10">
        <section className="w-full max-w-sm">
          <div className="mb-4 flex justify-end">
            <LanguageSwitcher />
          </div>
          <div className="mb-8 w-fit rounded-lg bg-brand-purple px-5 py-4 lg:hidden">
            <img
              src="/brand/21-mobile.png"
              alt="21 Mobile"
              className="h-10 w-auto"
            />
          </div>
          <h2 className="text-3xl font-bold">{t('auth.loginTitle')}</h2>
          <p className="mt-3 text-sm text-slate-500">
            {t('auth.loginDescription')}
          </p>
          <form
            className="mt-8 space-y-5"
            noValidate
            onSubmit={form.handleSubmit((body) => mutation.mutate(body))}
          >
            <div>
              <label
                htmlFor={`${id}-email`}
                className="block text-sm font-semibold"
              >
                {t('common.email')}
              </label>
              <input
                id={`${id}-email`}
                autoComplete="username"
                type="email"
                placeholder={t('auth.emailPlaceholder')}
                aria-invalid={!!form.formState.errors.email}
                aria-describedby={
                  form.formState.errors.email ? `${id}-email-error` : undefined
                }
                className="form-control mt-2 w-full"
                {...form.register('email')}
              />
              {form.formState.errors.email ? (
                <p
                  id={`${id}-email-error`}
                  role="alert"
                  className="mt-2 text-xs text-rose-700"
                >
                  {t('validation.email')}
                </p>
              ) : null}
            </div>
            <div>
              <label
                htmlFor={`${id}-password`}
                className="block text-sm font-semibold"
              >
                {t('auth.password')}
              </label>
              <div className="relative mt-2">
                <input
                  id={`${id}-password`}
                  autoComplete="current-password"
                  type={visible ? 'text' : 'password'}
                  aria-invalid={!!form.formState.errors.password}
                  aria-describedby={
                    form.formState.errors.password
                      ? `${id}-password-error`
                      : undefined
                  }
                  className="form-control w-full pr-12"
                  {...form.register('password')}
                />
                <button
                  type="button"
                  className="button-icon absolute right-0 top-0 rounded-l-none"
                  aria-label={
                    visible ? t('auth.hidePassword') : t('auth.showPassword')
                  }
                  aria-pressed={visible}
                  onClick={() => setVisible((current) => !current)}
                >
                  {visible ? (
                    <EyeOff size={18} aria-hidden="true" />
                  ) : (
                    <Eye size={18} aria-hidden="true" />
                  )}
                </button>
              </div>
              {form.formState.errors.password ? (
                <p
                  id={`${id}-password-error`}
                  role="alert"
                  className="mt-2 text-xs text-rose-700"
                >
                  {t('validation.passwordRequired')}
                </p>
              ) : null}
            </div>
            {factorRequired ? (
              <div className="space-y-3">
                <p className="text-sm font-semibold">
                  {t('auth.securityCode')}
                </p>
                {factorMode === 'app' ? (
                  <MfaCodeInput
                    value={factor}
                    onChange={(value) => form.setValue('factor', value)}
                    onComplete={(factor) => {
                      void form.handleSubmit((body) =>
                        mutation.mutate({ ...body, factor }),
                      )()
                    }}
                    disabled={mutation.isPending}
                  />
                ) : (
                  <input
                    id={`${id}-factor`}
                    className="form-control w-full font-mono"
                    type="text"
                    autoComplete="off"
                    maxLength={32}
                    placeholder={t('auth.recoveryCode')}
                    {...form.register('factor')}
                  />
                )}
                <button
                  type="button"
                  className="button-ghost"
                  onClick={() => {
                    form.setValue('factor', '')
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
            {mutation.isError ? (
              <p role="alert" className="text-sm text-rose-700">
                {isAxiosError(mutation.error) &&
                mutation.error.response?.data?.code === 'UNAUTHORIZED'
                  ? t('auth.invalidCredentials')
                  : errorMessage(mutation.error)}
              </p>
            ) : null}
            <label className="flex min-h-11 cursor-pointer items-center gap-3 text-sm text-slate-700">
              <input
                type="checkbox"
                className="h-4 w-4 accent-brand-purple"
                {...form.register('remember')}
              />
              {t('auth.rememberMe')}
            </label>
            <Link
              to="/esqueci-senha"
              className="block text-right text-sm font-semibold text-brand-purple underline-offset-4 hover:underline"
            >
              {t('auth.forgotPassword')}
            </Link>
            <button
              disabled={
                mutation.isPending ||
                (factorRequired &&
                  (factorMode === 'app'
                    ? !/^\d{6}$/.test(factor)
                    : !/^[0-9a-fA-F]{32}$/.test(factor)))
              }
              className="button-primary w-full"
            >
              {mutation.isPending ? (
                <LoaderCircle
                  size={17}
                  aria-hidden="true"
                  className="animate-spin"
                />
              ) : null}
              {mutation.isPending ? t('auth.signingIn') : t('auth.signIn')}
            </button>
          </form>
        </section>
      </main>
    </div>
  )
}
