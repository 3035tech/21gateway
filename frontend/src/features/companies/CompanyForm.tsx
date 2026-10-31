import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { companyFormSchema, saveCompany } from './api'
import type { Company, CompanyBody } from './api'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useTranslation } from 'react-i18next'
import { AppSelect } from '../../components/admin/AppSelect'
import { useState } from 'react'
import {
  CHANNEL_SMS,
  SMS_PROVIDER_ZOUG,
  canSaveProviderChannel,
  channelLabels,
  providerChannels,
  smsProviderOptions,
} from './providers'
import type { ChannelCode, ProviderCode } from './providers'
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
  const [selectedProvider, setSelectedProvider] = useState<ProviderCode>(
    (company?.smsProvider as ProviderCode) ?? SMS_PROVIDER_ZOUG,
  )
  const [channel, setChannel] = useState<ChannelCode>(CHANNEL_SMS)
  const availableChannels = providerChannels[selectedProvider] ?? [CHANNEL_SMS]
  const canSaveRoute = canSaveProviderChannel(selectedProvider, channel)
  return (
    <Dialog
      title={company ? t('companies.editTitle') : t('companies.new')}
      onClose={onClose}
      busy={mutation.isPending}
    >
      <form
        className="space-y-5"
        onSubmit={canSaveRoute
          ? form.handleSubmit((body) => mutation.mutate(body))
          : (event) => event.preventDefault()}
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
              value={selectedProvider}
              onValueChange={(value) => {
                setSelectedProvider(value as ProviderCode)
                setChannel(CHANNEL_SMS)
              }}
              options={smsProviderOptions}
            />
            <span className="mb-1 mt-4 block text-sm font-semibold text-slate-800">
              {t('companies.channel')}
            </span>
            <AppSelect
              label={t('companies.channel')}
              value={channel}
              onValueChange={(value) => setChannel(value as ChannelCode)}
              options={availableChannels.map((value) => ({
                value,
                label: channelLabels[value],
              }))}
            />
            <p className="mt-1 text-xs text-slate-600">
              {t(canSaveRoute ? 'companies.smsProviderHint' : 'companies.channelPending')}
            </p>
          </div>
          {selectedProvider === SMS_PROVIDER_ZOUG ? (
            <>
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
            </>
          ) : null}
        </div>
        {selectedProvider === SMS_PROVIDER_ZOUG && !clearProviderToken ? (
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
        {selectedProvider === SMS_PROVIDER_ZOUG && company?.providerTokenConfigured ? (
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
          disabled={!canSaveRoute}
          error={mutation.error}
          onCancel={onClose}
          submitLabel={company ? t('common.save') : t('companies.create')}
        />
      </form>
    </Dialog>
  )
}
