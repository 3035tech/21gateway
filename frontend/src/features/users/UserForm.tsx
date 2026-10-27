import { useForm, useWatch } from 'react-hook-form'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { userFormSchema, saveUser } from './api'
import type { SystemUser, UserBody } from './api'
import { RolePicker } from './RolePicker'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import { useTranslation } from 'react-i18next'
export function UserForm({
  user,
  onClose,
}: {
  user?: SystemUser
  onClose: () => void
}) {
  const { t } = useTranslation()
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
      title={user ? t('users.editTitle') : t('users.new')}
      onClose={onClose}
      busy={mutation.isPending}
    >
      <form
        className="space-y-5"
        onSubmit={form.handleSubmit((body) => mutation.mutate(body))}
      >
        <p className="text-sm leading-6 text-slate-600">
          {user ? t('users.editDescription') : t('users.inviteDescription')}
        </p>
        <fieldset disabled={mutation.isPending} className="space-y-5">
          <label className="field">
            {t('common.name')}
            <input
              className="form-control"
              autoComplete="name"
              {...form.register('name')}
            />
            <FieldError message={form.formState.errors.name?.message} />
          </label>
          <label className="field">
            {t('common.email')}
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
            {t('roles.globalAccess')}
          </p>
          <label className="flex min-h-11 cursor-pointer items-center gap-3 rounded-xl border border-slate-200 px-4 py-3 text-sm font-semibold text-slate-900">
            <input
              type="checkbox"
              className="h-4 w-4 accent-brand-purple"
              {...form.register('active')}
            />
            {t('users.activeAccess')}
          </label>
        </fieldset>
        <p className="text-xs leading-5 text-slate-500">
          {user
            ? t('users.editHint')
            : active
              ? t('users.inviteActiveHint')
              : t('users.inviteInactiveHint')}
        </p>
        <FormFooter
          pending={mutation.isPending}
          error={mutation.error}
          onCancel={onClose}
          submitLabel={user ? t('common.save') : t('users.sendInvitation')}
          pendingLabel={
            user ? t('common.saving') : t('users.sendingInvitation')
          }
        />
      </form>
    </Dialog>
  )
}
