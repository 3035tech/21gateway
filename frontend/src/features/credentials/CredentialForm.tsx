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
export function CredentialForm({
  credential,
  onClose,
  onSecret,
}: {
  credential?: Credential
  onClose: () => void
  onSecret: (username: string, password: string) => void
}) {
  const [search, setSearch] = useState('')
  const companies = useQuery({
    queryKey: ['admin', 'company-options', search],
    enabled: !credential?.companyId,
    queryFn: ({ signal }) =>
      listCompanies(
        { page: 1, pageSize: 100, search, direction: 'asc', sort: 'name' },
        signal,
      ),
  })
  const form = useForm<CredentialBody>({
    resolver: zodResolver(credentialFormSchema),
    defaultValues: {
      companyId: credential?.companyId ?? 0,
      username: credential?.username ?? '',
      active: credential?.active ?? true,
      callbackUrl: credential?.callbackUrl ?? '',
      callbacksEnabled: credential?.callbacksEnabled ?? false,
      preserveLegacyPassword: false,
      legacyPassword: '',
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
    if (result.password) onSecret(result.credential.username, result.password)
  }, onClose)

  const [preservePassword, removeToken, restrictIPs, active] = useWatch({
    control: form.control,
    name: [
      'preserveLegacyPassword',
      'clearProviderToken',
      'whitelistEnabled',
      'active',
    ],
  })
  return (
    <Dialog
      title={credential ? 'Editar credencial' : 'Nova credencial da API'}
      onClose={onClose}
      busy={mutation.isPending}
      wide
    >
      <p className="mb-6 max-w-xl text-sm leading-6 text-slate-600">
        Configure o acesso do cliente, o envio de mensagens e os destinos das
        notificações.
      </p>
      <form onSubmit={form.handleSubmit((body) => mutation.mutate(body))}>
        <fieldset disabled={mutation.isPending} className="min-w-0 space-y-7">
          <CredentialSection title="Acesso do cliente" icon={Building2}>
            <div className="grid gap-5 sm:grid-cols-2">
              <div className="space-y-4">
                {!credential?.companyId ? (
                  <label className="field">
                    Buscar empresa
                    <input
                      className="form-control"
                      placeholder="Nome ou referência"
                      value={search}
                      onChange={(event) => setSearch(event.target.value)}
                    />
                  </label>
                ) : null}
                <label className="field">
                  Empresa
                  <select
                    className="form-control"
                    disabled={
                      !!credential?.companyId ||
                      companies.isPending ||
                      companies.isError
                    }
                    {...form.register('companyId', { valueAsNumber: true })}
                  >
                    <option value={0}>
                      {companies.isPending && !credential?.companyId
                        ? 'Carregando empresas…'
                        : 'Selecione uma empresa'}
                    </option>
                    {credential?.companyId &&
                    !companies.data?.items.some(
                      (c) => c.id === credential.companyId,
                    ) ? (
                      <option value={credential.companyId}>
                        {credential.companyName}
                      </option>
                    ) : null}
                    {companies.data?.items.map((company) => (
                      <option value={company.id} key={company.id}>
                        {company.name}
                        {company.active ? '' : ' (bloqueada)'}
                      </option>
                    ))}
                  </select>
                  <FieldError
                    message={form.formState.errors.companyId?.message}
                  />
                </label>
                {companies.isError ? (
                  <p role="alert" className="text-xs text-rose-700">
                    Não foi possível carregar as empresas.{' '}
                    <button
                      className="underline underline-offset-2"
                      type="button"
                      onClick={() => void companies.refetch()}
                    >
                      Tentar novamente
                    </button>
                  </p>
                ) : null}
              </div>
              <div className="space-y-4">
                <label className="field">
                  Usuário da API
                  <input
                    className="form-control"
                    autoComplete="off"
                    readOnly={!!credential}
                    {...form.register('username')}
                  />
                  <FieldError
                    message={form.formState.errors.username?.message}
                  />
                </label>
                <label className="field">
                  ID legado{' '}
                  <span className="-mt-1 text-xs font-normal text-slate-500">
                    Opcional
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
            {!credential ? (
              <div className="mt-5 space-y-4 border-t border-slate-100 pt-4">
                <label className="flex min-h-11 items-center gap-3 text-sm font-medium">
                  <input
                    className="size-4 accent-brand-purple"
                    type="checkbox"
                    {...form.register('preserveLegacyPassword')}
                  />
                  Manter a senha de um cliente existente
                </label>
                {preservePassword ? (
                  <label className="field max-w-sm">
                    Senha atual da API 21
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
                      O cliente mantém seu acesso. A senha não será exibida após
                      salvar.
                    </span>
                  </label>
                ) : (
                  <p className="text-xs leading-5 text-slate-500">
                    Uma nova senha será gerada e exibida uma única vez após o
                    cadastro.
                  </p>
                )}
              </div>
            ) : null}
            <label className="mt-4 flex min-h-11 items-center gap-3 text-sm font-medium">
              <input
                className="size-4 accent-brand-purple"
                type="checkbox"
                {...form.register('active')}
              />
              Credencial ativa
            </label>
            {!active ? (
              <p
                role="status"
                className="mt-2 rounded-lg bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-900"
              >
                Ao salvar, os envios e as notificações desta credencial ficam
                bloqueados.
              </p>
            ) : null}
          </CredentialSection>
          <CredentialSection title="Envio pela Zoug" icon={KeyRound}>
            <p className="mb-4 text-sm leading-6 text-slate-600">
              {credential?.providerTokenConfigured
                ? 'Token configurado. Preencha apenas para substituir o atual.'
                : 'Adicione o token da conta do cliente para habilitar os envios.'}
            </p>
            {!removeToken ? (
              <label className="field">
                {credential?.providerTokenConfigured
                  ? 'Novo token'
                  : 'Token da Zoug'}
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
                  Remover o token desta conta
                </label>
                {removeToken ? (
                  <p
                    role="status"
                    className="rounded-lg bg-rose-50 px-3 py-2 text-xs leading-5 text-rose-800"
                  >
                    Ao salvar, novos envios ficam indisponíveis até configurar
                    outro token.
                  </p>
                ) : null}
              </div>
            ) : null}
          </CredentialSection>
          <CredentialSection title="Notificações ao cliente" icon={Link2}>
            <p className="mb-4 text-sm leading-6 text-slate-600">
              Use destinos separados para os status de entrega e as respostas
              recebidas.
            </p>
            <div className="grid gap-5 sm:grid-cols-2">
              <fieldset className="min-w-0 rounded-xl border border-slate-200 p-4">
                <legend className="px-1 text-sm font-semibold">
                  Status de entrega · DLR
                </legend>
                <label className="flex min-h-11 items-center gap-3 text-sm">
                  <input
                    className="size-4 accent-brand-purple"
                    type="checkbox"
                    {...form.register('callbacksEnabled')}
                  />
                  Encaminhar status
                </label>
                <label className="field mt-2">
                  URL de status (DLR)
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
                  O cliente recebe os status por GET nesta URL.
                </p>
              </fieldset>
              <fieldset className="min-w-0 rounded-xl border border-slate-200 p-4">
                <legend className="px-1 text-sm font-semibold">
                  Respostas recebidas · MO
                </legend>
                <label className="flex min-h-11 items-center gap-3 text-sm">
                  <input
                    className="size-4 accent-brand-purple"
                    type="checkbox"
                    {...form.register('moCallbacksEnabled')}
                  />
                  Encaminhar respostas
                </label>
                <label className="field mt-2">
                  URL de respostas (MO)
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
                  O cliente recebe as respostas por POST nesta URL.
                </p>
              </fieldset>
            </div>
            <p className="mt-4 text-xs leading-5 text-slate-500">
              Use URLs HTTPS. A entrega depende da homologação da integração com
              o fornecedor.
            </p>
          </CredentialSection>
          <CredentialSection title="Restrição de acesso" icon={ShieldCheck}>
            <label className="flex min-h-11 items-center gap-3 text-sm font-medium">
              <input
                className="size-4 accent-brand-purple"
                type="checkbox"
                {...form.register('whitelistEnabled')}
              />
              Permitir apenas IPs autorizados
            </label>
            {restrictIPs ? (
              <label className="field mt-3">
                IPs ou redes autorizadas
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
                  Um IP ou rede por linha. A restrição vale para chamadas à API
                  desta credencial.
                </span>
              </label>
            ) : (
              <p className="mt-2 text-xs leading-5 text-slate-500">
                O acesso não será limitado por IP. Usuário e senha continuam
                obrigatórios.
              </p>
            )}
          </CredentialSection>
        </fieldset>
        <div className="sticky -bottom-6 z-10 mt-7 bg-white pb-1 pt-3">
          <FormFooter
            pending={mutation.isPending}
            error={mutation.error}
            onCancel={onClose}
            submitLabel={credential ? 'Salvar alterações' : 'Criar credencial'}
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
