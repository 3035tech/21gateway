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

const schema = z
  .object({
    currentPassword: z.string().min(1, 'Informe a senha atual.').max(128),
    newPassword: z
      .string()
      .min(15, 'Use pelo menos 15 caracteres.')
      .max(128)
      .refine(
        (value) =>
          Array.from(value.trim()).length >=
          systemPasswordLimits.minimumCharacters,
        'Use pelo menos 15 caracteres sem contar espaços nas pontas.',
      ),
    confirmation: z.string(),
  })
  .refine((value) => value.newPassword === value.confirmation, {
    path: ['confirmation'],
    message: 'As senhas não coincidem.',
  })
  .refine((value) => value.currentPassword !== value.newPassword, {
    path: ['newPassword'],
    message: 'Escolha uma senha diferente da atual.',
  })
type FormValues = z.infer<typeof schema>

export function ProfilePage() {
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
    },
  })
  if (!user) return null
  return (
    <div className="max-w-4xl space-y-6">
      <div>
        <div className="flex items-center gap-3 text-brand-purple">
          <CircleUserRound size={30} aria-hidden="true" />
          <span className="text-xs font-bold uppercase tracking-[0.18em]">
            Conta e segurança
          </span>
        </div>
        <h1 className="mt-3 text-3xl font-bold tracking-tight text-slate-900">
          Meu perfil
        </h1>
        <p className="mt-2 text-sm text-slate-500">
          Confira seus dados de acesso e proteja sua conta administrativa.
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
          <CircleUserRound size={20} aria-hidden="true" /> Dados da conta
        </h2>
        <dl className="mt-5 grid gap-5 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-slate-500">Nome</dt>
            <dd className="mt-1 font-semibold text-slate-900">{user.name}</dd>
          </div>
          <div>
            <dt className="text-slate-500">E-mail</dt>
            <dd className="mt-1 font-semibold text-slate-900">{user.email}</dd>
          </div>
          <div>
            <dt className="text-slate-500">Perfil</dt>
            <dd className="mt-1 font-semibold text-slate-900">
              {roleLabels[user.role]}
            </dd>
          </div>
          <div>
            <dt className="text-slate-500">Conta criada em</dt>
            <dd className="mt-1 font-semibold text-slate-900">
              {formatDate(user.createdAt)}
            </dd>
          </div>
        </dl>
      </section>
      <section
        className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
        aria-labelledby="change-password"
      >
        <h2
          id="change-password"
          className="flex items-center gap-2 text-lg font-bold"
        >
          <KeyRound size={20} aria-hidden="true" /> Alterar senha
        </h2>
        <p className="mt-2 text-sm text-slate-500">
          Depois da alteração, todas as sessões serão encerradas. Entre
          novamente com a nova senha.
        </p>
        <form
          className="mt-5 max-w-lg space-y-4"
          onSubmit={form.handleSubmit((value) => mutation.mutate(value))}
        >
          <label className="field">
            Senha atual
            <input
              className="form-control"
              type="password"
              autoComplete="current-password"
              {...form.register('currentPassword')}
            />
            <span className="text-xs text-rose-700">
              {form.formState.errors.currentPassword?.message}
            </span>
          </label>
          <label className="field">
            Nova senha
            <input
              className="form-control"
              type="password"
              autoComplete="new-password"
              {...form.register('newPassword')}
            />
            <span className="text-xs text-rose-700">
              {form.formState.errors.newPassword?.message}
            </span>
          </label>
          <label className="field">
            Confirmar nova senha
            <input
              className="form-control"
              type="password"
              autoComplete="new-password"
              {...form.register('confirmation')}
            />
            <span className="text-xs text-rose-700">
              {form.formState.errors.confirmation?.message}
            </span>
          </label>
          <p className="text-xs leading-5 text-slate-500">
            Use uma frase com pelo menos{' '}
            {systemPasswordLimits.minimumCharacters} caracteres, sem contar
            espaços nas pontas. Evite senhas comuns.
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
            {mutation.isPending ? 'Alterando…' : 'Salvar nova senha'}
          </button>
        </form>
      </section>
      <MFASettings />
      {changed ? (
        <p
          role="status"
          className="rounded-xl bg-emerald-50 p-4 text-sm font-medium text-emerald-800"
        >
          Senha alterada. Entre novamente para continuar.
        </p>
      ) : null}
    </div>
  )
}
