import { useForm, useWatch } from 'react-hook-form'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { userFormSchema, saveUser } from './api'
import type { SystemUser, UserBody } from './api'
import { roles } from '../auth/api'
import { roleDescriptions, roleLabels } from '../../lib/admin'
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
        <label className="field">
          Nome
          <input className="form-control" {...form.register('name')} />
          <FieldError message={form.formState.errors.name?.message} />
        </label>
        <label className="field">
          E-mail
          <input
            className="form-control"
            type="email"
            {...form.register('email')}
          />
          <FieldError message={form.formState.errors.email?.message} />
        </label>
        <label className="field">
          Perfil
          <select className="form-control" {...form.register('role')}>
            {roles.map((role) => (
              <option key={role} value={role}>
                {roleLabels[role]}
              </option>
            ))}
          </select>
        </label>
        <p className="rounded-lg border border-purple-100 bg-purple-50 px-4 py-3 text-sm leading-6 text-purple-950">
          {roleDescriptions[selectedRole ?? 'viewer']} Todos os perfis são da
          equipe interna e veem dados de todas as empresas.
        </p>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" {...form.register('active')} />
          Usuário ativo
        </label>
        <p className="text-xs leading-5 text-slate-500">
          {user
            ? 'Alterar perfil ou status encerra as sessões desse usuário.'
            : 'Enviaremos um convite por e-mail para o usuário definir a própria senha. O link expira em 15 minutos.'}
        </p>
        <FormFooter
          pending={mutation.isPending}
          error={mutation.error}
          onCancel={onClose}
        />
      </form>
    </Dialog>
  )
}
