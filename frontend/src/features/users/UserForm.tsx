import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { userFormSchema, saveUser, systemPasswordLimits } from './api'
import type { SystemUser, UserBody } from './api'
import { roles } from '../auth/api'
import { roleLabels } from '../../lib/admin'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
export function UserForm({
  user,
  onClose,
}: {
  user?: SystemUser
  onClose: () => void
}) {
  const schema = user
    ? userFormSchema
    : userFormSchema.refine(
        (value) =>
          Array.from(value.password.trim()).length >=
          systemPasswordLimits.minimumCharacters,
        {
          path: ['password'],
          message: `Informe uma senha com pelo menos ${systemPasswordLimits.minimumCharacters} caracteres sem contar espaços nas pontas.`,
        },
      )
  const form = useForm<UserBody>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: user?.name ?? '',
      email: user?.email ?? '',
      role: user?.role ?? 'admin',
      active: user?.active ?? true,
      password: '',
    },
  })
  const mutation = useAdminMutation<UserBody>(
    (body) => saveUser(user?.id, body),
    onClose,
  )
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
        <label className="field">
          {user ? 'Nova senha (opcional)' : 'Senha'}
          <input
            className="form-control"
            type="password"
            autoComplete="new-password"
            {...form.register('password')}
          />
          <p className="text-xs text-slate-500">
            Use uma frase com pelo menos{' '}
            {systemPasswordLimits.minimumCharacters} caracteres, sem contar
            espaços nas pontas. Evite senhas comuns.
          </p>
          <FieldError message={form.formState.errors.password?.message} />
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" {...form.register('active')} />
          Usuário ativo
        </label>
        <p className="text-xs leading-5 text-slate-500">
          Alterar senha, perfil ou status encerra as sessões desse usuário.
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
