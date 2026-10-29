import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { z } from 'zod'
import {
  CircleUserRound,
  KeyRound,
  ShieldCheck,
  LoaderCircle,
} from 'lucide-react'
import { useUser } from '../auth/context'
import { changeOwnPassword } from '../auth/api'
import { systemPasswordLimits } from './api'
import { errorMessage, formatDate, roleLabels } from '../../lib/admin'
import { MFASettings } from './MFASettings'
import { useTranslation } from 'react-i18next'
import { useFeedback } from '../../components/admin/feedbackContext'
import { FieldError } from '../../components/admin/FormFooter'

const schema = z
  .object({
    currentPassword: z.string().min(1, 'validation.currentPassword').max(128),
    newPassword: z
      .string()
      .min(15, 'validation.passwordMinimum')
      .max(128)
      .refine(
        (value) =>
          Array.from(value.trim()).length >=
          systemPasswordLimits.minimumCharacters,
        'validation.passwordMinimumTrimmed',
      ),
    confirmation: z.string(),
  })
  .refine((value) => value.newPassword === value.confirmation, {
    path: ['confirmation'],
    message: 'validation.passwordMismatch',
  })
  .refine((value) => value.currentPassword !== value.newPassword, {
    path: ['newPassword'],
    message: 'validation.passwordDifferent',
  })
type FormValues = z.infer<typeof schema>

export function ProfilePage() {
  const { t } = useTranslation()
  const feedback = useFeedback()
  const user = useUser()
  const client = useQueryClient()
  const [changed, setChanged] = useState(false)
  const form = useForm<FormValues>({ resolver: zodResolver(schema) })
  const mutation = useMutation({
    mutationFn: (value: FormValues) =>
      changeOwnPassword({
        currentPassword: value.currentPassword,
        newPassword: value.newPassword,
      }),
    onSuccess: () => {
      form.reset()
      setChanged(true)
      client.removeQueries({ queryKey: ['admin'] })
      client.setQueryData(['session'], null)
      feedback.success(t('feedback.passwordChanged'))
    },
    onError: (error) => feedback.error(errorMessage(error)),
  })
  if (!user) return null
  return (
    <div className="max-w-6xl space-y-6">
      <div>
        <div className="flex items-center gap-3 text-brand-purple">
          <CircleUserRound size={30} aria-hidden="true" />
          <span className="text-xs font-bold uppercase tracking-[0.18em]">
            {t('auth.accountSecurity')}
          </span>
        </div>
        <h1 className="mt-3 text-3xl font-bold tracking-tight text-slate-900">
          {t('nav.profile')}
        </h1>
        <p className="mt-2 text-sm text-slate-500">
          {t('profile.description')}
        </p>
      </div>
      <section
        className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
        aria-labelledby="profile-details"
      >
        <h2
          id="profile-details"
          className="flex items-center gap-2 text-lg font-bold"
        >
          <CircleUserRound size={20} aria-hidden="true" />{' '}
          {t('profile.accountDetails')}
        </h2>
        <dl className="mt-5 grid gap-x-6 gap-y-5 text-sm sm:grid-cols-2 xl:grid-cols-4">
          <div>
            <dt className="text-slate-500">{t('common.name')}</dt>
            <dd className="mt-1 font-semibold text-slate-900">{user.name}</dd>
          </div>
          <div>
            <dt className="text-slate-500">{t('common.email')}</dt>
            <dd className="mt-1 font-semibold text-slate-900">{user.email}</dd>
          </div>
          <div>
            <dt className="text-slate-500">{t('roles.label')}</dt>
            <dd className="mt-1 font-semibold text-slate-900">
              {t(roleLabels[user.role])}
            </dd>
          </div>
          <div>
            <dt className="text-slate-500">{t('profile.createdAt')}</dt>
            <dd className="mt-1 font-semibold text-slate-900">
              {formatDate(user.createdAt)}
            </dd>
          </div>
        </dl>
      </section>
      <div className="grid items-start gap-6 xl:grid-cols-2">
        <section
          className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
          aria-labelledby="change-password"
        >
          <h2
            id="change-password"
            className="flex items-center gap-2 text-lg font-bold"
          >
            <KeyRound size={20} aria-hidden="true" />{' '}
            {t('profile.changePassword')}
          </h2>
          <p className="mt-2 text-sm leading-6 text-slate-500">
            {t('profile.changePasswordHint')}
          </p>
          <form
            className="mt-5 space-y-4"
            onSubmit={form.handleSubmit((value) => mutation.mutate(value))}
          >
            <label className="field">
              {t('profile.currentPassword')}
              <input
                className="form-control"
                type="password"
                autoComplete="current-password"
                {...form.register('currentPassword')}
              />
              <FieldError
                message={form.formState.errors.currentPassword?.message}
              />
            </label>
            <label className="field">
              {t('auth.newPassword')}
              <input
                className="form-control"
                type="password"
                autoComplete="new-password"
                {...form.register('newPassword')}
              />
              <FieldError
                message={form.formState.errors.newPassword?.message}
              />
            </label>
            <label className="field">
              {t('profile.confirmNewPassword')}
              <input
                className="form-control"
                type="password"
                autoComplete="new-password"
                {...form.register('confirmation')}
              />
              <FieldError
                message={form.formState.errors.confirmation?.message}
              />
            </label>
            <p className="text-xs leading-5 text-slate-500">
              {t('profile.passwordHint', {
                count: systemPasswordLimits.minimumCharacters,
              })}
            </p>
            {mutation.isError ? (
              <p role="alert" className="text-sm text-rose-700">
                {errorMessage(mutation.error)}
              </p>
            ) : null}
            <button className="button-primary" disabled={mutation.isPending}>
              {mutation.isPending ? (
                <LoaderCircle
                  size={17}
                  className="animate-spin"
                  aria-hidden="true"
                />
              ) : (
                <ShieldCheck size={17} aria-hidden="true" />
              )}
              {mutation.isPending
                ? t('profile.changing')
                : t('auth.saveNewPassword')}
            </button>
          </form>
        </section>
        <MFASettings />
      </div>
      {changed ? (
        <p
          role="status"
          className="rounded-xl bg-emerald-50 p-4 text-sm font-medium text-emerald-800"
        >
          {t('profile.passwordChanged')}
        </p>
      ) : null}
    </div>
  )
}
