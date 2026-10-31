import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { companyFormSchema, saveCompany } from './api'
import type { Company, CompanyBody } from './api'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useTranslation } from 'react-i18next'
import { AppSelect } from '../../components/admin/AppSelect'
import { SMS_PROVIDER_ZOUG } from './providers'
export function CompanyForm({
  company,
  onClose,
}: {
  company?: Company
  onClose: () => void
}) {
  const { t } = useTranslation()
  const form = useForm<CompanyBody>({
    resolver: zodResolver(companyFormSchema),
    defaultValues: {
      providerToken: '',
      clearProviderToken: false,
      name: company?.name ?? '',
      reference: company?.reference ?? '',
      active: company?.active ?? true,
      smsProvider: SMS_PROVIDER_ZOUG,
    },
  })
  const mutation = useAdminMutation<CompanyBody>(
    (body) => saveCompany(company?.id, body),
    onClose,
    company ? 'feedback.updated' : 'feedback.created',
  )
  const clearProviderToken = useWatch({
    control: form.control,
    name: 'clearProviderToken',
  })
  const smsProvider = useWatch({ control: form.control, name: 'smsProvider' })
  return (
    <Dialog
      title={company ? t('companies.editTitle') : t('companies.new')}
      onClose={onClose}
      busy={mutation.isPending}
    >
      <form
        className="space-y-5"
        onSubmit={form.handleSubmit((body) => mutation.mutate(body))}
      >
        <label className="field">
          {t('common.name')}
          <input className="form-control" {...form.register('name')} />
          <FieldError message={form.formState.errors.name?.message} />
        </label>
        <label className="field">
          {t('common.reference')}
          <input className="form-control" {...form.register('reference')} />
          <FieldError message={form.formState.errors.reference?.message} />
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" {...form.register('active')} />
          {t('companies.active')}
        </label>
        <div className="rounded-xl border border-purple-100 bg-purple-50 p-4">
          <div className="mb-4">
            <span className="mb-1 block text-sm font-semibold text-slate-800">
              {t('companies.smsProvider')}
            </span>
            <AppSelect
              label={t('companies.smsProvider')}
              value={smsProvider ?? SMS_PROVIDER_ZOUG}
              onValueChange={(value) =>
                form.setValue('smsProvider', value as typeof SMS_PROVIDER_ZOUG)
              }
              options={[{ value: SMS_PROVIDER_ZOUG, label: 'Zoug' }]}
            />
            <p className="mt-1 text-xs text-slate-600">
              {t('companies.smsProviderHint')}
            </p>
          </div>
          <h3 className="font-bold text-brand-purple">
            {t('companies.providerAccess')}
          </h3>
          <p className="mt-1 text-sm leading-5 text-slate-600">
            {company?.providerTokenConfigured
              ? t('companies.providerConfigured')
              : company?.legacyProviderTokenFallback
                ? t('companies.providerLegacy')
                : t('companies.providerMissing')}
          </p>
        </div>
        {!clearProviderToken ? (
          <label className="field">
            {company?.providerTokenConfigured
              ? t('companies.providerReplace')
              : t('companies.providerToken')}
            <input
              type="password"
              autoComplete="new-password"
              className="form-control"
              {...form.register('providerToken')}
            />
            <span className="text-xs font-normal text-slate-500">
              {t(
                company
                  ? 'companies.providerIssuedHint'
                  : 'companies.providerNewHint',
              )}
            </span>
            <FieldError
              message={form.formState.errors.providerToken?.message}
            />
          </label>
        ) : null}
        {company?.providerTokenConfigured ? (
          <label className="flex min-h-11 items-center gap-2 text-sm text-rose-800">
            <input
              type="checkbox"
              {...form.register('clearProviderToken', {
                onChange: (event) => {
                  if (event.target.checked) form.setValue('providerToken', '')
                },
              })}
            />
            {t('companies.providerRemove')}
          </label>
        ) : null}
        <FormFooter
          pending={mutation.isPending}
          error={mutation.error}
          onCancel={onClose}
          submitLabel={company ? t('common.save') : t('companies.create')}
        />
      </form>
    </Dialog>
  )
}
