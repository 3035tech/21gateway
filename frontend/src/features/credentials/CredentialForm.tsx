import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { credentialFormSchema, saveCredential } from './api'
import type { Credential, CredentialBody } from './api'
import { listCompanies } from '../companies/api'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import { Building2, KeyRound, Link2, ShieldCheck } from 'lucide-react'
import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useTranslation } from 'react-i18next'
function localDateTime(value?: string | null) {
  if (!value) return ''
  const date = new Date(value)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
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
  const [passwordMode, setPasswordMode] = useState<
    'generated' | 'manual' | 'legacy'
  >('generated')
  const companies = useQuery({
    queryKey: ['admin', 'company-options', search],
    enabled: !selectedCompanyId,
    queryFn: ({ signal }) =>
      listCompanies(
        { page: 1, pageSize: 100, search, direction: 'asc', sort: 'name' },
        signal,
      ),
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
      legacyAccountId: credential?.legacyAccountId ?? '',
    },
  })
  const mutation = useAdminMutation<CredentialBody>(async (body) => {
    const result = await saveCredential(credential?.id, body)
    if (result.password || (!credential && body.generateUsername))
      onSecret(result.credential.username, result.password)
  }, onClose)

  const [generateUsername, removeToken, restrictIPs, active] = useWatch({
    control: form.control,
    name: [
      'generateUsername',
      'clearProviderToken',
      'whitelistEnabled',
      'active',
    ],
  })
  return (
    <Dialog
      title={
        credential ? t('credentials.editTitle') : t('credentials.newTitle')
      }
      onClose={onClose}
      busy={mutation.isPending}
      wide
    >
      <p className="mb-6 max-w-xl text-sm leading-6 text-slate-600">
        {t('credentials.formDescription')}
      </p>
      <form onSubmit={form.handleSubmit((body) => mutation.mutate(body))}>
        <fieldset disabled={mutation.isPending} className="min-w-0 space-y-7">
          <CredentialSection
            title={t('credentials.clientAccess')}
            icon={Building2}
          >
            <p className="mb-4 text-sm leading-6 text-slate-600">
              {t('credentials.basicAuthHint')}
            </p>
            <div className="grid gap-5 sm:grid-cols-2">
              <div className="space-y-4">
                {!selectedCompanyId ? (
                  <label className="field">
                    {t('credentials.searchCompany')}
                    <input
                      className="form-control"
                      placeholder={t('credentials.companySearchPlaceholder')}
                      value={search}
                      onChange={(event) => setSearch(event.target.value)}
                    />
                  </label>
                ) : null}
                <label className="field">
                  {t('common.company')}
                  <select
                    className="form-control"
                    disabled={
                      !!selectedCompanyId ||
                      companies.isPending ||
                      companies.isError
                    }
                    {...form.register('companyId', { valueAsNumber: true })}
                  >
                    <option value={0}>
                      {companies.isPending && !selectedCompanyId
                        ? t('credentials.loadingCompanies')
                        : t('credentials.selectCompany')}
                    </option>
                    {selectedCompanyId &&
                    !companies.data?.items.some(
                      (c) => c.id === selectedCompanyId,
                    ) ? (
                      <option value={selectedCompanyId}>
                        {credential?.companyName ?? initialCompanyName}
                      </option>
                    ) : null}
                    {companies.data?.items.map((company) => (
                      <option value={company.id} key={company.id}>
                        {company.name}
                        {company.active ? '' : t('credentials.blockedSuffix')}
                      </option>
                    ))}
                  </select>
                  <FieldError
                    message={form.formState.errors.companyId?.message}
                  />
                </label>
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
              <div className="space-y-4">
                {!credential ? (
                  <label className="flex min-h-11 items-center gap-3 text-sm font-medium">
                    <input
                      className="size-4 accent-brand-purple"
                      type="checkbox"
                      {...form.register('generateUsername')}
                    />
                    {t('credentials.generateUsername')}
                  </label>
                ) : null}
                <label className="field">
                  {t('common.apiUsername')}
                  <input
                    className="form-control"
                    autoComplete="off"
                    readOnly={!!credential || !!generateUsername}
                    placeholder={
                      generateUsername
                        ? t('credentials.generatedUsernamePlaceholder')
                        : undefined
                    }
                    {...form.register('username')}
                  />
                  <FieldError
                    message={form.formState.errors.username?.message}
                  />
                  {generateUsername ? (
                    <span className="text-xs font-normal leading-5 text-slate-500">
                      {t('credentials.generatedUsernameHint')}
                    </span>
                  ) : null}
                </label>
                <label className="field">
                  {t('credentials.legacyId')}{' '}
                  <span className="-mt-1 text-xs font-normal text-slate-500">
                    {t('credentials.optional')}
                  </span>
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
              <div className="mt-5 space-y-4 border-t border-slate-100 pt-4">
                <fieldset className="space-y-2">
                  <legend className="text-sm font-semibold">
                    {t('credentials.passwordSetup')}
                  </legend>
                  {(['generated', 'manual', 'legacy'] as const).map((mode) => (
                    <label
                      key={mode}
                      className="flex min-h-10 items-center gap-3 text-sm"
                    >
                      <input
                        type="radio"
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
                      {t(`credentials.passwordMode.${mode}`)}
                    </label>
                  ))}
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
                ) : (
                  <p className="text-xs leading-5 text-slate-500">
                    {t('credentials.newPasswordHint')}
                  </p>
                )}
              </div>
            )}
            <label className="mt-4 flex min-h-11 items-center gap-3 text-sm font-medium">
              <input
                className="size-4 accent-brand-purple"
                type="checkbox"
                {...form.register('active')}
              />
              {t('credentials.active')}
            </label>
            {!active ? (
              <p
                role="status"
                className="mt-2 rounded-lg bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-900"
              >
                {t('credentials.inactiveHint')}
              </p>
            ) : null}
            <label className="field mt-4 max-w-sm">
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
          </CredentialSection>
          <CredentialSection
            title={t('credentials.providerSection')}
            icon={KeyRound}
          >
            <p className="mb-4 text-sm leading-6 text-slate-600">
              {credential?.providerTokenConfigured
                ? t('credentials.tokenConfiguredHint')
                : t('credentials.tokenMissingHint')}
            </p>
            {!removeToken ? (
              <label className="field">
                {credential?.providerTokenConfigured
                  ? t('credentials.newToken')
                  : t('credentials.providerToken')}
                <input
                  type="password"
                  autoComplete="new-password"
                  className="form-control"
                  {...form.register('providerToken')}
                />
                <FieldError
                  message={form.formState.errors.providerToken?.message}
                />
              </label>
            ) : null}
            {credential?.providerTokenConfigured ? (
              <div className="mt-3">
                <label className="flex min-h-11 items-center gap-3 text-sm text-rose-800">
                  <input
                    className="size-4 accent-rose-700"
                    type="checkbox"
                    {...form.register('clearProviderToken')}
                    onChange={(event) => {
                      form.setValue(
                        'clearProviderToken',
                        event.target.checked,
                        { shouldDirty: true, shouldValidate: true },
                      )
                      if (event.target.checked)
                        form.setValue('providerToken', '', {
                          shouldDirty: true,
                          shouldValidate: true,
                        })
                    }}
                  />
                  {t('credentials.removeToken')}
                </label>
                {removeToken ? (
                  <p
                    role="status"
                    className="rounded-lg bg-rose-50 px-3 py-2 text-xs leading-5 text-rose-800"
                  >
                    {t('credentials.removeTokenHint')}
                  </p>
                ) : null}
              </div>
            ) : null}
          </CredentialSection>
          <CredentialSection
            title={t('credentials.notificationsSection')}
            icon={Link2}
          >
            <p className="mb-4 text-sm leading-6 text-slate-600">
              {t('credentials.notificationsHint')}
            </p>
            <div className="grid gap-5 sm:grid-cols-2">
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
              </fieldset>
            </div>
            <p className="mt-4 text-xs leading-5 text-slate-500">
              {t('credentials.httpsHint')}
            </p>
          </CredentialSection>
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
              <label className="field mt-3">
                {t('credentials.allowedIps')}
                <textarea
                  rows={3}
                  spellCheck={false}
                  className="form-control font-mono"
                  placeholder={'203.0.113.10\n198.51.100.0/24'}
                  {...form.register('allowedIPsText')}
                />
                <FieldError
                  message={form.formState.errors.allowedIPsText?.message}
                />
                <span className="text-xs font-normal leading-5 text-slate-500">
                  {t('credentials.ipHint')}
                </span>
              </label>
            ) : (
              <p className="mt-2 text-xs leading-5 text-slate-500">
                {t('credentials.noIpHint')}
              </p>
            )}
          </CredentialSection>
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
