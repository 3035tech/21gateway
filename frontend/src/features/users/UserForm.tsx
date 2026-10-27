import { useForm, useWatch } from 'react-hook-form'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { userFormSchema, saveUser } from './api'
import type { SystemUser, UserBody } from './api'
import { RolePicker } from './RolePicker'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
export function UserForm({
  user,
  onClose,
}: {
  user?: SystemUser
  onClose: () => void
}) {
  const client = useQueryClient()
  const form = useForm<UserBody>({
    resolver: zodResolver(userFormSchema),
    defaultValues: {
      name: user?.name ?? '',
      email: user?.email ?? '',
      role: user?.role ?? 'viewer',
      active: user?.active ?? true,
    },
  })
  const selectedRole = useWatch({ control: form.control, name: 'role' })
  const active = useWatch({ control: form.control, name: 'active' })
  const mutation = useMutation({
    mutationFn: (body: UserBody) => saveUser(user?.id, body),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['admin'] })
      await client.invalidateQueries({ queryKey: ['session'] })
      onClose()
    },
    onError: () => {
      if (!user) void client.invalidateQueries({ queryKey: ['admin'] })
    },
  })
  return (
    <Dialog
      title={user ? 'Editar usuário' : 'Novo usuário'}
      onClose={onClose}
      busy={mutation.isPending}
    >
      <form
        className="space-y-5"
        onSubmit={form.handleSubmit((body) => mutation.mutate(body))}
      >
        <p className="text-sm leading-6 text-slate-600">
          {user
            ? 'Atualize os dados e o acesso deste usuário interno.'
            : 'Convide um novo usuário para o painel. Ele definirá a própria senha pelo e-mail recebido.'}
        </p>
        <fieldset disabled={mutation.isPending} className="space-y-5">
          <label className="field">
            Nome
            <input
              className="form-control"
              autoComplete="name"
              {...form.register('name')}
            />
            <FieldError message={form.formState.errors.name?.message} />
          </label>
          <label className="field">
            E-mail
            <input
              className="form-control"
              type="email"
              autoComplete="email"
              {...form.register('email')}
            />
            <FieldError message={form.formState.errors.email?.message} />
          </label>
          <RolePicker
            value={selectedRole ?? 'viewer'}
            disabled={mutation.isPending}
            onChange={(role) =>
              form.setValue('role', role, {
                shouldDirty: true,
                shouldValidate: true,
              })
            }
          />
          <p className="text-xs leading-5 text-slate-500">
            Estes perfis são da equipe interna e têm visibilidade de todas as
            empresas.
          </p>
          <label className="flex min-h-11 cursor-pointer items-center gap-3 rounded-xl border border-slate-200 px-4 py-3 text-sm font-semibold text-slate-900">
            <input
              type="checkbox"
              className="h-4 w-4 accent-brand-purple"
              {...form.register('active')}
            />
            Acesso ativo
          </label>
        </fieldset>
        <p className="text-xs leading-5 text-slate-500">
          {user
            ? 'Alterar o perfil ou o status encerra as sessões atuais deste usuário.'
            : active
              ? 'O convite expira em 15 minutos. O usuário poderá entrar após definir a senha.'
              : 'O convite expira em 15 minutos. O usuário só poderá entrar depois que o acesso for ativado.'}
        </p>
        <FormFooter
          pending={mutation.isPending}
          error={mutation.error}
          onCancel={onClose}
          submitLabel={user ? 'Salvar alterações' : 'Enviar convite'}
          pendingLabel={user ? 'Salvando…' : 'Enviando convite…'}
        />
      </form>
    </Dialog>
  )
}
