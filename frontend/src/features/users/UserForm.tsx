import { useForm, useWatch } from 'react-hook-form'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { userFormSchema, saveUser } from './api'
import type { SystemUser, UserBody } from './api'
import { RolePicker } from './RolePicker'
import { CompanyAccessPicker } from './CompanyAccessPicker'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import { useTranslation } from 'react-i18next'
import { useFeedback } from '../../components/admin/feedbackContext'
import { errorMessage } from '../../lib/admin'
import { useEffect, useState } from 'react'
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
  const feedback = useFeedback()
  const client = useQueryClient()
  const [companySearch, setCompanySearch] = useState('')
  const [debouncedCompanySearch, setDebouncedCompanySearch] = useState('')
  const [companyPage, setCompanyPage] = useState(1)
  useEffect(() => {
    const timer = window.setTimeout(
      () => setDebouncedCompanySearch(companySearch),
      250,
    )
    return () => window.clearTimeout(timer)
  }, [companySearch])
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
    queryKey: [
      'admin',
      'user-company-options',
      debouncedCompanySearch,
      companyPage,
    ],
    enabled: selectedRole !== 'admin',
    queryFn: ({ signal }) =>
      listCompanies(
        {
          page: companyPage,
          pageSize: 20,
          search: debouncedCompanySearch,
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
      feedback.success(t(user ? 'feedback.updated' : 'feedback.inviteSent'))
    },
    onError: (error) => {
      feedback.error(errorMessage(error))
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
              <CompanyAccessPicker
                options={(companyOptions.data?.items ?? []).map((company) => ({
                  id: String(company.id),
                  name: company.name,
                }))}
                selected={companyIds.map((id, index) => ({
                  id: String(id),
                  name:
                    selectedCompanies[index]?.data?.name ??
                    t('common.companyNumber', { id }),
                }))}
                search={companySearch}
                onSearchChange={(value) => {
                  setCompanySearch(value)
                  setCompanyPage(1)
                }}
                onToggle={(value) => {
                  const id = Number(value)
                  form.setValue(
                    'companyIds',
                    companyIds.includes(id)
                      ? companyIds.filter((current) => current !== id)
                      : [...companyIds, id],
                    { shouldValidate: true, shouldDirty: true },
                  )
                }}
                page={companyPage}
                pageSize={20}
                total={companyOptions.data?.total ?? 0}
                onPageChange={setCompanyPage}
                loading={
                  companySearch !== debouncedCompanySearch ||
                  companyOptions.isPending ||
                  companyOptions.isFetching
                }
                error={companyOptions.isError}
                onRetry={() => void companyOptions.refetch()}
              />
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
