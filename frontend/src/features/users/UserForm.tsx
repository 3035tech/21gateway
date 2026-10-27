import { useForm, useWatch } from 'react-hook-form'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { userFormSchema, saveUser } from './api'
import type { SystemUser, UserBody } from './api'
import { RolePicker } from './RolePicker'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { useQueries, useQuery } from '@tanstack/react-query'
import { getCompany, listCompanies } from '../companies/api'
export function UserForm({
  user,
  initialCompanyId,
  onClose,
}: {
  user?: SystemUser
  initialCompanyId?: number
  onClose: () => void
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [companySearch, setCompanySearch] = useState('')
  const form = useForm<UserBody>({
    resolver: zodResolver(userFormSchema),
    defaultValues: {
      name: user?.name ?? '',
      email: user?.email ?? '',
      role: user?.role ?? 'viewer',
      active: user?.active ?? true,
      companyIds:
        user?.companyIds ?? (initialCompanyId ? [initialCompanyId] : []),
    },
  })
  const selectedRole = useWatch({ control: form.control, name: 'role' })
  const active = useWatch({ control: form.control, name: 'active' })
  const companyIds =
    useWatch({ control: form.control, name: 'companyIds' }) ?? []
  const companyOptions = useQuery({
    queryKey: ['admin', 'user-company-options', companySearch],
    enabled: selectedRole !== 'admin',
    queryFn: ({ signal }) =>
      listCompanies(
        {
          page: 1,
          pageSize: 100,
          search: companySearch,
          sort: 'name',
          direction: 'asc',
          active: true,
        },
        signal,
      ),
  })
  const selectedCompanies = useQueries({
    queries: companyIds.map((id) => ({
      queryKey: ['admin', 'company', id],
      queryFn: ({ signal }: { signal: AbortSignal }) => getCompany(id, signal),
    })),
  })
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
            onChange={(role) => {
              if (role === 'admin')
                form.setValue('companyIds', [], {
                  shouldValidate: true,
                  shouldDirty: true,
                })
              form.setValue('role', role, {
                shouldDirty: true,
                shouldValidate: true,
              })
            }}
          />
          <p className="text-xs leading-5 text-slate-500">
            {selectedRole === 'admin'
              ? t('roles.globalAccess')
              : t('users.companyAccessHint')}
          </p>
          {selectedRole !== 'admin' ? (
            <div className="space-y-3">
              <label className="field">
                {t('users.companyAccess')}
                <input
                  className="form-control"
                  type="search"
                  value={companySearch}
                  onChange={(event) => setCompanySearch(event.target.value)}
                  placeholder={t('companies.searchPlaceholder')}
                />
              </label>
              <div className="max-h-40 overflow-y-auto rounded-lg border border-slate-200">
                {companyOptions.data?.items
                  .filter((company) => !companyIds.includes(company.id))
                  .map((company) => (
                    <button
                      key={company.id}
                      className="flex min-h-11 w-full items-center px-3 text-left text-sm hover:bg-purple-50"
                      type="button"
                      onClick={() =>
                        form.setValue(
                          'companyIds',
                          [...companyIds, company.id],
                          { shouldValidate: true, shouldDirty: true },
                        )
                      }
                    >
                      {company.name}
                    </button>
                  ))}
                {companyOptions.isPending ? (
                  <p className="p-3 text-sm text-slate-500">
                    {t('common.loading')}
                  </p>
                ) : null}
              </div>
              <ul className="flex flex-wrap gap-2">
                {companyIds.map((id, index) => (
                  <li
                    key={id}
                    className="flex items-center gap-2 rounded-lg bg-purple-50 px-3 py-1 text-sm text-brand-purple"
                  >
                    {selectedCompanies[index]?.data?.name ??
                      t('common.companyNumber', { id })}
                    <button
                      type="button"
                      className="table-action-danger min-h-8 px-1"
                      aria-label={t('users.removeCompany', { id })}
                      onClick={() =>
                        form.setValue(
                          'companyIds',
                          companyIds.filter((value) => value !== id),
                          { shouldValidate: true, shouldDirty: true },
                        )
                      }
                    >
                      ×
                    </button>
                  </li>
                ))}
              </ul>
              <FieldError message={form.formState.errors.companyIds?.message} />
            </div>
          ) : null}
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
