import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Controller, useForm, useWatch } from 'react-hook-form'
import type { UseFormReturn } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { credentialFormSchema, saveCredential } from './api'
import type { Credential, CredentialBody } from './api'
import { getCompany, listCompanies } from '../companies/api'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import {
  ArrowUpRight,
  Building2,
  KeyRound,
  Link2,
  ShieldCheck,
} from 'lucide-react'
import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { AppSelect } from '../../components/admin/AppSelect'
import { AllowedIPsEditor } from './AllowedIPsEditor'

type CredentialTab = 'access' | 'notifications' | 'ips'
const credentialTabs: CredentialTab[] = ['access', 'notifications', 'ips']
function localDateTime(value?: string | null) {
  if (!value) return ''
  const date = new Date(value)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}
function CallbackAuthFields({
  form,
  prefix,
}: {
  form: UseFormReturn<CredentialBody>
  prefix: 'callbackAuth' | 'moCallbackAuth'
}) {
  const { t } = useTranslation()
  const mode = useWatch({ control: form.control, name: `${prefix}.mode` })
  const configured = useWatch({
    control: form.control,
    name: `${prefix}.secretConfigured`,
  })
  const existingMode = useWatch({
    control: form.control,
    name: `${prefix}.existingMode`,
  })
  return (
    <div className="mt-5 border-t border-slate-200 pt-5">
      <label className="field">
        {t('credentials.callbackAuthMode')}
        <Controller
          name={`${prefix}.mode`}
          control={form.control}
          render={({ field }) => (
            <AppSelect
              value={field.value}
              label={t('credentials.callbackAuthMode')}
              onValueChange={(value) => {
                field.onChange(value)
                form.setValue(`${prefix}.secret`, '')
                form.clearErrors(prefix)
              }}
              options={[
                { value: 'none', label: t('credentials.callbackAuthNone') },
                { value: 'header', label: t('credentials.callbackAuthHeader') },
                { value: 'basic', label: t('credentials.callbackAuthBasic') },
              ]}
            />
          )}
        />
      </label>
      {mode !== 'none' ? (
        <div className="mt-4 grid gap-4 sm:grid-cols-2 sm:items-start">
          {mode === 'header' ? (
            <label className="field min-w-0">
              {t('credentials.callbackHeaderName')}
              <input
                className="form-control w-full"
                autoComplete="off"
                placeholder="X-Client-Token"
                {...form.register(`${prefix}.headerName`)}
              />
              <FieldError
                message={form.formState.errors[prefix]?.headerName?.message}
              />
            </label>
          ) : (
            <label className="field min-w-0">
              {t('credentials.callbackBasicUsername')}
              <input
                className="form-control w-full"
                autoComplete="off"
                {...form.register(`${prefix}.username`)}
              />
              <FieldError
                message={form.formState.errors[prefix]?.username?.message}
              />
            </label>
          )}
          <label className="field min-w-0">
            {mode === 'basic'
              ? t('credentials.callbackBasicPassword')
              : t('credentials.callbackHeaderValue')}
            <input
              className="form-control w-full"
              type="password"
              autoComplete="new-password"
              {...form.register(`${prefix}.secret`)}
            />
            <FieldError
              message={form.formState.errors[prefix]?.secret?.message}
            />
            <span className="text-xs font-normal leading-5 text-slate-500">
              {configured && mode === existingMode
                ? t('credentials.callbackSecretSaved')
                : t('credentials.callbackSecretNew')}
            </span>
          </label>
        </div>
      ) : null}
    </div>
  )
}
export function CredentialForm({
  credential,
  initialCompanyId,
  initialCompanyName,
  onClose,
  onSecret,
}: {
  credential?: Credential
  initialCompanyId?: number
  initialCompanyName?: string
  onClose: () => void
  onSecret: (username: string, password: string) => void
}) {
  const { t } = useTranslation()
  const selectedCompanyId = credential?.companyId ?? initialCompanyId
  const [search, setSearch] = useState('')
  const [companyPage, setCompanyPage] = useState(1)
  const [selectedCompany, setSelectedCompany] = useState<{
    id: number
    publicRef: string
    name: string
    active: boolean
  } | null>(null)
  const [passwordMode, setPasswordMode] = useState<
    'generated' | 'manual' | 'legacy'
  >('generated')
  const [activeTab, setActiveTab] = useState<CredentialTab>('access')
  const [legacyOpen, setLegacyOpen] = useState(!!credential?.legacyAccountId)
  const [ipDraftValid, setIpDraftValid] = useState(true)
  const companies = useQuery({
    queryKey: ['admin', 'company-options', search, companyPage],
    enabled: !selectedCompanyId,
    queryFn: ({ signal }) =>
      listCompanies(
        {
          page: companyPage,
          pageSize: 20,
          search,
          direction: 'asc',
          sort: 'name',
        },
        signal,
      ),
  })
  const fixedCompany = useQuery({
    queryKey: ['admin', 'company', selectedCompanyId],
    enabled: !!selectedCompanyId,
    queryFn: ({ signal }) => getCompany(selectedCompanyId!, signal),
  })
  const form = useForm<CredentialBody>({
    resolver: zodResolver(credentialFormSchema),
    defaultValues: {
      companyId: selectedCompanyId ?? 0,
      generateUsername: !credential,
      passwordMode: 'generated',
      username: credential?.username ?? '',
      active: credential?.active ?? true,
      callbackUrl: credential?.callbackUrl ?? '',
      callbacksEnabled: credential?.callbacksEnabled ?? false,
      callbackAuth: {
        mode: credential?.callbackAuth?.mode ?? 'none',
        headerName: credential?.callbackAuth?.headerName ?? '',
        username: credential?.callbackAuth?.username ?? '',
        secret: '',
        secretConfigured: credential?.callbackAuth?.secretConfigured ?? false,
        existingMode: credential?.callbackAuth?.mode ?? 'none',
      },
      preserveLegacyPassword: false,
      legacyPassword: '',
      manualPassword: '',
      expiresAtText: localDateTime(credential?.expiresAt),
      providerToken: '',
      clearProviderToken: false,
      whitelistEnabled: credential?.whitelistEnabled ?? false,
      allowedIPsText: credential?.allowedIPs.join('\n') ?? '',
      moCallbackUrl: credential?.moCallbackUrl ?? '',
      moCallbacksEnabled: credential?.moCallbacksEnabled ?? false,
      moCallbackAuth: {
        mode: credential?.moCallbackAuth?.mode ?? 'none',
        headerName: credential?.moCallbackAuth?.headerName ?? '',
        username: credential?.moCallbackAuth?.username ?? '',
        secret: '',
        secretConfigured: credential?.moCallbackAuth?.secretConfigured ?? false,
        existingMode: credential?.moCallbackAuth?.mode ?? 'none',
      },
      legacyAccountId: credential?.legacyAccountId ?? '',
    },
  })
  const mutation = useAdminMutation<CredentialBody>(
    async (body) => {
      const result = await saveCredential(credential?.id, body)
      if (result.password || (!credential && body.generateUsername))
        onSecret(result.credential.username, result.password)
    },
    onClose,
    credential ? 'feedback.updated' : 'feedback.created',
  )

  const [generateUsername, restrictIPs, active, formCompanyId] = useWatch({
    control: form.control,
    name: ['generateUsername', 'whitelistEnabled', 'active', 'companyId'],
  })
  const companyPublicRef =
    selectedCompany?.publicRef ??
    fixedCompany.data?.publicRef ??
    companies.data?.items.find((company) => company.id === formCompanyId)
      ?.publicRef
  const errors = form.formState.errors
  const tabErrors: Record<CredentialTab, boolean> = {
    access: !!(
      errors.companyId ||
      errors.username ||
      errors.legacyAccountId ||
      errors.legacyPassword ||
      errors.manualPassword ||
      errors.expiresAtText ||
      errors.providerToken
    ),
    notifications: !!(
      errors.callbackUrl ||
      errors.moCallbackUrl ||
      errors.callbackAuth ||
      errors.moCallbackAuth
    ),
    ips: !!errors.allowedIPsText,
  }
  function selectTab(tab: CredentialTab) {
    setActiveTab(tab)
    const dialog = document
      .getElementById(`credential-tab-${tab}`)
      ?.closest('dialog')
    if (dialog) dialog.scrollTop = 0
  }
  return (
    <Dialog
      title={
        credential ? t('credentials.editTitle') : t('credentials.newTitle')
      }
      onClose={onClose}
      busy={mutation.isPending}
      wide
    >
      <p className="mb-6 max-w-2xl text-sm leading-6 text-slate-600">
        {t(
          credential
            ? 'credentials.editFormDescription'
            : 'credentials.formDescription',
        )}
      </p>
      <form
        noValidate
        onSubmit={(event) => {
          if (restrictIPs && !ipDraftValid) {
            event.preventDefault()
            selectTab('ips')
            document.getElementById('allowed-ip-entry')?.focus()
            return
          }
          void form.handleSubmit(
            (body) => mutation.mutate(body),
            (invalid) => {
              const tab: CredentialTab =
                invalid.companyId ||
                invalid.username ||
                invalid.legacyAccountId ||
                invalid.legacyPassword ||
                invalid.manualPassword ||
                invalid.expiresAtText ||
                invalid.providerToken
                  ? 'access'
                  : invalid.callbackUrl ||
                      invalid.moCallbackUrl ||
                      invalid.callbackAuth ||
                      invalid.moCallbackAuth
                    ? 'notifications'
                    : 'ips'
              selectTab(tab)
              document.getElementById(`credential-tab-${tab}`)?.focus()
            },
          )(event)
        }}
      >
        <div
          role="tablist"
          aria-label={t('credentials.formSections')}
          className="sticky top-0 z-10 mb-6 grid grid-cols-3 gap-1 border-b border-slate-200 bg-white pb-2"
        >
          {credentialTabs.map((tab) => {
            const Icon =
              tab === 'access'
                ? Building2
                : tab === 'notifications'
                  ? Link2
                  : ShieldCheck
            return (
              <button
                key={tab}
                id={`credential-tab-${tab}`}
                type="button"
                role="tab"
                aria-selected={activeTab === tab}
                aria-controls={`credential-panel-${tab}`}
                tabIndex={activeTab === tab ? 0 : -1}
                disabled={mutation.isPending}
                className={`flex min-w-0 items-center justify-center gap-2 rounded-lg border-b-2 px-2 py-3 text-xs font-semibold sm:text-sm ${activeTab === tab ? 'border-brand-purple bg-purple-50 text-brand-purple' : 'border-transparent text-slate-600 hover:bg-slate-50'}`}
                onClick={() => selectTab(tab)}
                onKeyDown={(event) => {
                  if (
                    !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(
                      event.key,
                    )
                  )
                    return
                  event.preventDefault()
                  const index = credentialTabs.indexOf(tab)
                  const next =
                    event.key === 'Home'
                      ? 0
                      : event.key === 'End'
                        ? credentialTabs.length - 1
                        : (index +
                            (event.key === 'ArrowRight' ? 1 : -1) +
                            credentialTabs.length) %
                          credentialTabs.length
                  const nextTab = credentialTabs[next]
                  selectTab(nextTab)
                  document.getElementById(`credential-tab-${nextTab}`)?.focus()
                }}
              >
                <Icon size={16} aria-hidden="true" className="shrink-0" />
                <span className="truncate">{t(`credentials.tabs.${tab}`)}</span>
                {tabErrors[tab] ? (
                  <span
                    className="size-2 shrink-0 rounded-full bg-rose-600"
                    aria-label={t('credentials.tabHasError')}
                  />
                ) : null}
              </button>
            )
          })}
        </div>
        <fieldset disabled={mutation.isPending} className="min-w-0">
          <div
            id="credential-panel-access"
            role="tabpanel"
            aria-labelledby="credential-tab-access"
            hidden={activeTab !== 'access'}
            className="space-y-7"
          >
            <CredentialSection
              title={t('credentials.clientAccess')}
              icon={Building2}
            >
              <p className="mb-4 text-sm leading-6 text-slate-600 sm:mb-5">
                {t(
                  credential
                    ? 'credentials.editBasicAuthHint'
                    : 'credentials.basicAuthHint',
                )}
              </p>
              {!selectedCompanyId ? (
                <label className="field mb-5">
                  {t('credentials.searchCompany')}
                  <input
                    className="form-control"
                    placeholder={t('credentials.companySearchPlaceholder')}
                    value={search}
                    onChange={(event) => {
                      setSearch(event.target.value)
                      setCompanyPage(1)
                    }}
                  />
                </label>
              ) : null}
              {!credential ? (
                <label className="mb-5 flex min-h-11 items-center gap-3 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm font-medium text-slate-800">
                  <input
                    className="size-4 shrink-0 accent-brand-purple"
                    type="checkbox"
                    {...form.register('generateUsername')}
                  />
                  {t('credentials.generateUsername')}
                </label>
              ) : null}
              <div className="grid gap-5 sm:grid-cols-2">
                <div>
                  {selectedCompanyId ? (
                    <>
                      <input
                        type="hidden"
                        value={selectedCompanyId}
                        {...form.register('companyId', { valueAsNumber: true })}
                      />
                      <label className="field">
                        {t('common.company')}
                        <input
                          className="form-control"
                          readOnly
                          value={
                            credential?.companyName ??
                            initialCompanyName ??
                            fixedCompany.data?.name ??
                            t('common.companyNumber', { id: selectedCompanyId })
                          }
                        />
                      </label>
                    </>
                  ) : (
                    <div className="field">
                      <span>{t('common.company')}</span>
                      <Controller
                        name="companyId"
                        control={form.control}
                        render={({ field }) => (
                          <AppSelect
                            value={field.value}
                            label={t('common.company')}
                            disabled={companies.isPending || companies.isError}
                            onValueChange={(value) => {
                              const id = Number(value)
                              field.onChange(id)
                              setSelectedCompany(
                                companies.data?.items.find(
                                  (company) => company.id === id,
                                ) ?? null,
                              )
                            }}
                            options={[
                              {
                                value: 0,
                                label: companies.isPending
                                  ? t('credentials.loadingCompanies')
                                  : t('credentials.selectCompany'),
                              },
                              ...[
                                ...(selectedCompany ? [selectedCompany] : []),
                                ...(companies.data?.items ?? []),
                              ]
                                .filter(
                                  (company, index, items) =>
                                    items.findIndex(
                                      (item) => item.id === company.id,
                                    ) === index,
                                )
                                .map((company) => ({
                                  value: company.id,
                                  label: `${company.name}${company.active ? '' : t('credentials.blockedSuffix')}`,
                                })),
                            ]}
                          />
                        )}
                      />
                      <FieldError
                        message={form.formState.errors.companyId?.message}
                      />
                      {(companies.data?.total ?? 0) > 20 ? (
                        <span className="flex items-center justify-between gap-2 text-xs font-normal text-slate-600">
                          <button
                            type="button"
                            className="button-ghost min-h-9 px-2"
                            disabled={companyPage <= 1 || companies.isFetching}
                            onClick={() => setCompanyPage((page) => page - 1)}
                          >
                            {t('common.previous')}
                          </button>
                          {t('users.companyPage', {
                            page: companyPage,
                            pages: Math.ceil((companies.data?.total ?? 0) / 20),
                          })}
                          <button
                            type="button"
                            className="button-ghost min-h-9 px-2"
                            disabled={
                              companyPage * 20 >=
                                (companies.data?.total ?? 0) ||
                              companies.isFetching
                            }
                            onClick={() => setCompanyPage((page) => page + 1)}
                          >
                            {t('common.next')}
                          </button>
                        </span>
                      ) : null}
                    </div>
                  )}
                  {companies.isError ? (
                    <p role="alert" className="text-xs text-rose-700">
                      {t('credentials.companyLoadFailed')}{' '}
                      <button
                        className="rounded px-1 font-semibold text-brand-purple underline underline-offset-2 hover:bg-purple-50"
                        type="button"
                        onClick={() => void companies.refetch()}
                      >
                        {t('common.retry')}
                      </button>
                    </p>
                  ) : null}
                </div>
                <div>
                  {generateUsername ? (
                    <div className="field">
                      <span>{t('credentials.basicUsername')}</span>
                      <div className="form-control flex items-center bg-slate-50 text-slate-500">
                        {t('credentials.generatedUsernamePlaceholder')}
                      </div>
                      <span className="text-xs font-normal leading-5 text-slate-500">
                        {t('credentials.generatedUsernameHint')}
                      </span>
                    </div>
                  ) : (
                    <label className="field">
                      {t('credentials.basicUsername')}
                      <input
                        className="form-control"
                        aria-label={t('credentials.basicUsername')}
                        autoComplete="off"
                        data-lpignore="true"
                        data-1p-ignore="true"
                        readOnly={!!credential}
                        {...form.register('username')}
                      />
                      <FieldError
                        message={form.formState.errors.username?.message}
                      />
                    </label>
                  )}
                </div>
              </div>
              {credential ? (
                <div className="mt-5 rounded-xl border border-slate-200 bg-slate-50 p-4">
                  <p className="text-sm font-semibold text-slate-900">
                    {t('credentials.passwordSetup')}
                  </p>
                  <p className="mt-1 text-sm leading-6 text-slate-600">
                    {t('credentials.passwordStoredHint')}
                  </p>
                </div>
              ) : (
                <div className="mt-6 space-y-4 border-t border-slate-200 pt-6">
                  <fieldset>
                    <legend className="mb-3 text-base font-bold text-slate-900">
                      {t('credentials.passwordSetup')}
                    </legend>
                    <div className="grid gap-3 sm:grid-cols-2">
                      {(['generated', 'manual', 'legacy'] as const).map(
                        (mode) => (
                          <label
                            key={mode}
                            className={`flex cursor-pointer items-start gap-3 rounded-xl border p-4 text-sm transition-colors ${mode === 'legacy' ? 'sm:col-span-2' : ''} ${passwordMode === mode ? 'border-brand-purple bg-purple-50/70' : 'border-slate-200 bg-white hover:border-slate-300'}`}
                          >
                            <input
                              type="radio"
                              className="mt-0.5 size-4 shrink-0 accent-brand-purple"
                              name="credential-password-mode"
                              value={mode}
                              checked={passwordMode === mode}
                              onChange={() => {
                                setPasswordMode(mode)
                                form.setValue('passwordMode', mode)
                                form.setValue(
                                  'preserveLegacyPassword',
                                  mode === 'legacy',
                                )
                                form.setValue('legacyPassword', '')
                                form.setValue('manualPassword', '')
                              }}
                            />
                            <span className="space-y-1">
                              <span className="block font-bold leading-5 text-slate-900">
                                {t(`credentials.passwordMode.${mode}`)}
                              </span>
                              <span className="block text-xs leading-5 text-slate-600">
                                {t(`credentials.passwordModeHint.${mode}`)}
                              </span>
                            </span>
                          </label>
                        ),
                      )}
                    </div>
                  </fieldset>
                  {passwordMode === 'legacy' ? (
                    <label className="field max-w-sm">
                      {t('credentials.currentApiPassword')}
                      <input
                        type="password"
                        autoComplete="new-password"
                        className="form-control"
                        {...form.register('legacyPassword')}
                      />
                      <FieldError
                        message={form.formState.errors.legacyPassword?.message}
                      />
                      <span className="text-xs font-normal leading-5 text-slate-500">
                        {t('credentials.preservePasswordHint')}
                      </span>
                    </label>
                  ) : passwordMode === 'manual' ? (
                    <label className="field max-w-sm">
                      {t('credentials.manualPassword')}
                      <input
                        type="password"
                        autoComplete="new-password"
                        className="form-control"
                        {...form.register('manualPassword')}
                      />
                      <FieldError
                        message={form.formState.errors.manualPassword?.message}
                      />
                      <span className="text-xs font-normal leading-5 text-slate-500">
                        {t('credentials.manualPasswordHint')}
                      </span>
                    </label>
                  ) : null}
                  <p className="text-xs leading-5 text-slate-600">
                    {t('credentials.passwordStorageHint')}
                  </p>
                </div>
              )}
              <div className="mt-6 grid gap-4 border-t border-slate-200 pt-6 sm:grid-cols-2 sm:items-start">
                <div className="field">
                  <span>{t('common.status')}</span>
                  <label className="form-control flex items-center gap-3 text-sm font-semibold">
                    <input
                      className="size-4 shrink-0 accent-brand-purple"
                      type="checkbox"
                      {...form.register('active')}
                    />
                    {t('credentials.active')}
                  </label>
                  {!active ? (
                    <p
                      role="status"
                      className="rounded-lg bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-900"
                    >
                      {t('credentials.inactiveHint')}
                    </p>
                  ) : (
                    <p className="text-xs font-normal leading-5 text-slate-500">
                      {t('credentials.activeHint')}
                    </p>
                  )}
                </div>
                <label className="field">
                  {t('credentials.expiresAt')}
                  <input
                    type="datetime-local"
                    className="form-control"
                    {...form.register('expiresAtText')}
                  />
                  <FieldError
                    message={form.formState.errors.expiresAtText?.message}
                  />
                  <span className="text-xs font-normal leading-5 text-slate-500">
                    {t('credentials.expiryHint')}
                  </span>
                </label>
              </div>
              <details
                className="mt-6 rounded-xl border border-slate-200 px-4 py-3"
                open={legacyOpen || !!errors.legacyAccountId}
                onToggle={(event) => setLegacyOpen(event.currentTarget.open)}
              >
                <summary className="cursor-pointer text-sm font-semibold text-slate-800">
                  {t('credentials.legacySettings')}
                </summary>
                <p className="mt-3 text-xs leading-5 text-slate-600">
                  {t('credentials.legacySettingsHint')}
                </p>
                <label className="field mt-3 max-w-sm">
                  {t('credentials.legacyId')}
                  <input
                    className="form-control"
                    readOnly={!!credential?.legacyAccountId}
                    placeholder="2340 ou 2340_779"
                    {...form.register('legacyAccountId')}
                  />
                  <FieldError
                    message={form.formState.errors.legacyAccountId?.message}
                  />
                </label>
              </details>
            </CredentialSection>
            <div className="rounded-2xl border border-purple-100 bg-purple-50/70 p-4 sm:p-5">
              <div className="flex items-start gap-3">
                <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-white text-brand-purple shadow-sm">
                  <KeyRound size={19} aria-hidden="true" />
                </span>
                <div className="min-w-0">
                  <h3 className="font-bold text-slate-900">
                    {t('credentials.providerSection')}
                  </h3>
                  <p className="mt-1 text-sm leading-6 text-slate-600">
                    {t('credentials.providerAtCompany')}
                  </p>
                </div>
              </div>
              {companyPublicRef ? (
                <Link
                  to={`/empresas/${companyPublicRef}`}
                  onClick={onClose}
                  className="mt-4 inline-flex min-h-10 items-center gap-1 rounded-lg px-2 text-sm font-bold text-brand-purple underline-offset-4 hover:bg-white hover:underline"
                >
                  {t('credentials.openCompanyProvider')}
                  <ArrowUpRight size={16} aria-hidden="true" />
                </Link>
              ) : null}
            </div>
          </div>
          <div
            id="credential-panel-notifications"
            role="tabpanel"
            aria-labelledby="credential-tab-notifications"
            hidden={activeTab !== 'notifications'}
          >
            <CredentialSection
              title={t('credentials.notificationsSection')}
              icon={Link2}
            >
              <p className="mb-4 text-sm leading-6 text-slate-600">
                {t('credentials.notificationsHint')}
              </p>
              <div className="grid gap-5">
                <fieldset className="min-w-0 rounded-xl border border-slate-200 p-4">
                  <legend className="px-1 text-sm font-semibold">
                    {t('credentials.dlrTitle')}
                  </legend>
                  <label className="flex min-h-11 items-center gap-3 text-sm">
                    <input
                      className="size-4 accent-brand-purple"
                      type="checkbox"
                      {...form.register('callbacksEnabled')}
                    />
                    {t('credentials.forwardStatus')}
                  </label>
                  <label className="field mt-2">
                    {t('credentials.dlrUrl')}
                    <input
                      className="form-control w-full"
                      type="url"
                      autoComplete="off"
                      placeholder="https://cliente.com.br/status"
                      {...form.register('callbackUrl')}
                    />
                    <FieldError
                      message={form.formState.errors.callbackUrl?.message}
                    />
                  </label>
                  <p className="mt-3 text-xs leading-5 text-slate-500">
                    {t('credentials.dlrHint')}
                  </p>
                  <CallbackAuthFields form={form} prefix="callbackAuth" />
                </fieldset>
                <fieldset className="min-w-0 rounded-xl border border-slate-200 p-4">
                  <legend className="px-1 text-sm font-semibold">
                    {t('credentials.moTitle')}
                  </legend>
                  <label className="flex min-h-11 items-center gap-3 text-sm">
                    <input
                      className="size-4 accent-brand-purple"
                      type="checkbox"
                      {...form.register('moCallbacksEnabled')}
                    />
                    {t('credentials.forwardReplies')}
                  </label>
                  <label className="field mt-2">
                    {t('credentials.moUrl')}
                    <input
                      className="form-control w-full"
                      type="url"
                      autoComplete="off"
                      placeholder="https://cliente.com.br/respostas"
                      {...form.register('moCallbackUrl')}
                    />
                    <FieldError
                      message={form.formState.errors.moCallbackUrl?.message}
                    />
                  </label>
                  <p className="mt-3 text-xs leading-5 text-slate-500">
                    {t('credentials.moHint')}
                  </p>
                  <CallbackAuthFields form={form} prefix="moCallbackAuth" />
                </fieldset>
              </div>
              <p className="mt-4 text-xs leading-5 text-slate-500">
                {t('credentials.httpsHint')}
              </p>
            </CredentialSection>
          </div>
          <div
            id="credential-panel-ips"
            role="tabpanel"
            aria-labelledby="credential-tab-ips"
            hidden={activeTab !== 'ips'}
          >
            <CredentialSection
              title={t('credentials.accessRestriction')}
              icon={ShieldCheck}
            >
              <label className="flex min-h-11 items-center gap-3 text-sm font-medium">
                <input
                  className="size-4 accent-brand-purple"
                  type="checkbox"
                  {...form.register('whitelistEnabled')}
                />
                {t('credentials.whitelistEnabled')}
              </label>
              {restrictIPs ? (
                <div className="mt-4">
                  <Controller
                    name="allowedIPsText"
                    control={form.control}
                    render={({ field }) => (
                      <AllowedIPsEditor
                        value={field.value ?? ''}
                        onChange={field.onChange}
                        onValidityChange={setIpDraftValid}
                      />
                    )}
                  />
                  <FieldError
                    message={form.formState.errors.allowedIPsText?.message}
                  />
                </div>
              ) : (
                <p className="mt-2 text-xs leading-5 text-slate-500">
                  {t('credentials.noIpHint')}
                </p>
              )}
            </CredentialSection>
          </div>
        </fieldset>
        <div className="sticky -bottom-6 z-10 mt-7 bg-white pb-1 pt-3">
          <FormFooter
            pending={mutation.isPending}
            error={mutation.error}
            onCancel={onClose}
            submitLabel={
              credential ? t('common.save') : t('credentials.create')
            }
          />
        </div>
      </form>
    </Dialog>
  )
}
function CredentialSection({
  title,
  icon: Icon,
  children,
}: {
  title: string
  icon: LucideIcon
  children: ReactNode
}) {
  return (
    <fieldset className="min-w-0 border-t border-slate-200 pt-4">
      <legend className="flex items-center gap-2 pr-3 text-sm font-bold text-brand-purple">
        <Icon size={17} aria-hidden="true" />
        {title}
      </legend>
      {children}
    </fieldset>
  )
}
