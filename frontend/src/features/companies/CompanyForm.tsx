import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { companyFormSchema, saveCompany } from './api'
import type { Company, CompanyBody, PlannedRoute } from './api'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useTranslation } from 'react-i18next'
import { AppSelect } from '../../components/admin/AppSelect'
import { useState } from 'react'
import { ChevronDown, Info, Plus, X } from 'lucide-react'
import {
  CHANNEL_SMS,
  PROVIDER_GOOGLE,
  SMS_PROVIDER_ZOUG,
  providerLabel,
  canPlanProviderChannel,
  canSaveProviderChannel,
  channelLabels,
  providerChannels,
  providerOptions,
  ROUTE_PLANNED,
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
  const activeSMSProvider =
    (company?.smsProvider as ProviderCode) ?? SMS_PROVIDER_ZOUG
  const activeSMSLabel = providerLabel(activeSMSProvider)
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
  const [selectedProvider, setSelectedProvider] =
    useState<ProviderCode>(activeSMSProvider)
  const [channel, setChannel] = useState<ChannelCode>(
    providerChannels[activeSMSProvider]?.[0] ?? CHANNEL_SMS,
  )
  const [plannedRoutes, setPlannedRoutes] = useState<PlannedRoute[]>(() =>
    (company?.providerRoutes ?? [])
      .filter(
        (route) =>
          route.status === ROUTE_PLANNED &&
          canPlanProviderChannel(route.provider as ProviderCode, route.channel),
      )
      .map((route) => ({
        channel: route.channel as PlannedRoute['channel'],
        provider: route.provider as PlannedRoute['provider'],
      })),
  )
  const [plannedRoutesChanged, setPlannedRoutesChanged] = useState(false)
  const availableChannels = providerChannels[selectedProvider] ?? [CHANNEL_SMS]
  const isActiveSelection = canSaveProviderChannel(selectedProvider, channel)
  const canPlanSelection = canPlanProviderChannel(selectedProvider, channel)
  const plannedForChannel = plannedRoutes.find(
    (route) => route.channel === channel,
  )
  const addPlannedRoute = () => {
    if (!canPlanSelection) return
    setPlannedRoutesChanged(true)
    setPlannedRoutes((current) => [
      ...current.filter((route) => route.channel !== channel),
      {
        channel: channel as PlannedRoute['channel'],
        provider: selectedProvider as PlannedRoute['provider'],
      },
    ])
  }
  return (
    <Dialog
      title={company ? t('companies.editTitle') : t('companies.new')}
      onClose={onClose}
      busy={mutation.isPending}
    >
      <form
        className="space-y-5"
        onSubmit={form.handleSubmit((body) =>
          mutation.mutate({
            ...body,
            ...(plannedRoutesChanged ? { plannedRoutes } : {}),
          }),
        )}
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
        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
          <h3 className="text-sm font-bold text-slate-900">
            {t('companies.routesTitle')}
          </h3>
          <p className="mt-1 text-xs text-slate-600">
            {t('companies.routesHint')}
          </p>
          <div className="mt-4 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-emerald-100 bg-white px-3 py-2 text-sm">
            <span className="min-w-0 font-semibold text-slate-800">
              SMS · {activeSMSLabel}
            </span>
            <span className="rounded-full bg-emerald-50 px-2 py-1 text-xs font-semibold text-emerald-800">
              {t('companies.routeActive')}
            </span>
          </div>
          {plannedRoutes.map((route) => (
            <div
              key={route.channel}
              className="mt-2 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-amber-100 bg-white px-3 py-2 text-sm"
            >
              <div className="min-w-0">
                <span className="break-words font-semibold text-slate-800">
                  {channelLabels[route.channel]} ·{' '}
                  {
                    providerOptions.find(
                      (option) => option.value === route.provider,
                    )?.label
                  }
                </span>
                {route.provider === PROVIDER_GOOGLE ? (
                  <p className="mt-1 text-xs leading-5 text-slate-600">
                    {t('companies.googleRCSRouteHint')}
                  </p>
                ) : null}
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <span className="rounded-full bg-amber-50 px-2 py-1 text-xs font-semibold text-amber-800">
                  {t('companies.routePlanned')}
                </span>
                <button
                  type="button"
                  className="flex size-11 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-purple"
                  aria-label={t('companies.removePlannedRoute', {
                    channel: channelLabels[route.channel],
                  })}
                  onClick={() => {
                    setPlannedRoutesChanged(true)
                    setPlannedRoutes((current) =>
                      current.filter((item) => item.channel !== route.channel),
                    )
                  }}
                >
                  <X size={16} aria-hidden="true" />
                </button>
              </div>
            </div>
          ))}
        </div>
        <details className="group rounded-xl border border-purple-100 bg-purple-50 p-4">
          <summary className="flex min-h-11 cursor-pointer list-none items-center justify-between gap-3 rounded-lg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-purple [&::-webkit-details-marker]:hidden">
            <span className="text-sm font-bold text-slate-900">
              {t('companies.addRouteTitle')}
            </span>
            <ChevronDown
              size={18}
              aria-hidden="true"
              className="shrink-0 text-brand-purple transition-transform group-open:rotate-180"
            />
          </summary>
          <div className="mt-3 border-t border-purple-100 pt-4">
            <p className="mb-4 text-xs text-slate-600">
              {t('companies.addRouteHint')}
            </p>
            <span className="mb-1 block text-sm font-semibold text-slate-800">
              {t('companies.smsProvider')}
            </span>
            <AppSelect
              label={t('companies.smsProvider')}
              value={selectedProvider}
              onValueChange={(value) => {
                const provider = value as ProviderCode
                setSelectedProvider(provider)
                setChannel(providerChannels[provider][0])
              }}
              options={providerOptions}
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
            <p className="mt-2 text-xs font-semibold text-slate-700">
              {t(
                isActiveSelection
                  ? 'companies.selectionActive'
                  : canPlanSelection
                    ? 'companies.selectionPlanned'
                    : 'companies.selectionUnavailable',
              )}
            </p>
            {selectedProvider === PROVIDER_GOOGLE ? (
              <p className="mt-2 flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs leading-5 text-amber-900">
                <Info
                  size={16}
                  className="mt-0.5 shrink-0"
                  aria-hidden="true"
                />
                {t('companies.googleRCSPrerequisite')}
              </p>
            ) : (
              <p className="mt-1 text-xs text-slate-600">
                {t(
                  isActiveSelection
                    ? 'companies.smsProviderHint'
                    : canPlanSelection
                      ? 'companies.planRouteHint'
                      : 'companies.routeUnavailable',
                  { provider: activeSMSLabel },
                )}
              </p>
            )}
            {canPlanSelection ? (
              <button
                type="button"
                className="button-secondary mt-3 inline-flex w-full items-center justify-center gap-2 sm:w-auto"
                onClick={addPlannedRoute}
                disabled={plannedForChannel?.provider === selectedProvider}
              >
                <Plus size={16} aria-hidden="true" />
                {t(
                  plannedForChannel
                    ? 'companies.replacePlannedRoute'
                    : 'companies.addPlannedRoute',
                )}
              </button>
            ) : null}
          </div>
        </details>
        <div className="space-y-3">
          <h3 className="font-bold text-brand-purple">
            {t('companies.providerAccess')}
          </h3>
          <p className="text-sm leading-5 text-slate-600">
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
