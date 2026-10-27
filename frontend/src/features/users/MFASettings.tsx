import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'
import { ShieldCheck, ShieldOff, Copy, KeyRound } from 'lucide-react'
import { beginMFA, confirmMFA, disableMFA, mfaStatus } from '../auth/api'
import { MfaCodeInput } from '../auth/MfaCodeInput'
import { errorMessage } from '../../lib/admin'
import { useTranslation } from 'react-i18next'

export function MFASettings() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const status = useQuery({ queryKey: ['admin', 'mfa'], queryFn: mfaStatus })
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [factorMode, setFactorMode] = useState<'app' | 'recovery'>('app')
  const [setup, setSetup] = useState<{ secret: string; uri: string } | null>(
    null,
  )
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
  const begin = useMutation({
    mutationFn: () => beginMFA(password),
    onSuccess: setSetup,
  })
  const confirm = useMutation({
    mutationFn: (value: string) => confirmMFA(value),
    onSuccess: (value) => {
      setRecoveryCodes(value.recoveryCodes)
      setSetup(null)
      setPassword('')
      setCode('')
      client.setQueryData(['admin', 'mfa'], { enabled: true })
    },
  })
  const disable = useMutation({
    mutationFn: (value: string) => disableMFA(password, value),
    onSuccess: () => {
      setPassword('')
      setCode('')
      client.setQueryData(['admin', 'mfa'], { enabled: false })
    },
  })
  return (
    <section
      className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
      aria-labelledby="mfa-heading"
    >
      <h2
        id="mfa-heading"
        className="flex items-center gap-2 text-lg font-bold"
      >
        <ShieldCheck size={20} aria-hidden="true" /> {t('mfa.title')}
      </h2>
      <p className="mt-2 text-sm text-slate-500">{t('mfa.description')}</p>
      {status.isPending ? (
        <p role="status" className="mt-5 text-sm text-slate-500">
          {t('mfa.loading')}
        </p>
      ) : null}
      {status.isError ? (
        <p role="alert" className="mt-5 text-sm text-rose-700">
          {errorMessage(status.error)}
        </p>
      ) : null}
      {status.data ? (
        <div className="mt-5">
          <span
            className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-bold ${status.data.enabled ? 'bg-emerald-50 text-emerald-800' : 'bg-amber-50 text-amber-800'}`}
          >
            {status.data.enabled ? t('mfa.enabled') : t('mfa.disabled')}
          </span>
          {recoveryCodes.length > 0 ? (
            <div className="mt-5 rounded-xl border border-emerald-200 bg-emerald-50 p-5">
              <h3 className="font-bold text-emerald-950">
                {t('mfa.recoveryCodes')}
              </h3>
              <p className="mt-2 text-sm text-emerald-900">
                {t('mfa.recoveryHint')}
              </p>
              <div className="mt-4 grid gap-2 font-mono text-sm sm:grid-cols-2">
                {recoveryCodes.map((value) => (
                  <code key={value}>{value}</code>
                ))}
              </div>
              <button
                type="button"
                className="mt-4 inline-flex min-h-11 items-center gap-2 rounded-lg border border-emerald-300 bg-white px-4 text-sm font-semibold text-emerald-900 transition-colors hover:bg-emerald-100 active:bg-emerald-200"
                onClick={() =>
                  void navigator.clipboard.writeText(recoveryCodes.join('\n'))
                }
              >
                <Copy size={16} aria-hidden="true" /> {t('mfa.copyCodes')}
              </button>
              <button
                type="button"
                className="ml-3 min-h-11 rounded-lg px-3 text-sm font-semibold text-emerald-900 underline transition-colors hover:bg-emerald-100 active:bg-emerald-200"
                onClick={() => setRecoveryCodes([])}
              >
                {t('mfa.savedCodes')}
              </button>
            </div>
          ) : null}
          {!status.data.enabled && !setup ? (
            <form
              className="mt-5 max-w-md space-y-3"
              onSubmit={(event) => {
                event.preventDefault()
                begin.mutate()
              }}
            >
              <label className="field">
                {t('mfa.confirmPassword')}
                <input
                  className="form-control"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </label>
              {begin.isError ? (
                <p role="alert" className="text-sm text-rose-700">
                  {errorMessage(begin.error)}
                </p>
              ) : null}
              <button
                type="submit"
                disabled={begin.isPending}
                className="button-primary"
              >
                <KeyRound size={16} aria-hidden="true" />{' '}
                {t('mfa.configureApp')}
              </button>
            </form>
          ) : null}
          {setup ? (
            <div className="mt-5 space-y-4">
              <p className="text-sm font-medium">{t('mfa.scanHint')}</p>
              <div className="w-fit rounded-xl border border-slate-200 bg-white p-4">
                <QRCodeSVG
                  value={setup.uri}
                  size={192}
                  marginSize={4}
                  title={t('mfa.qrTitle')}
                />
              </div>
              <div>
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                  {t('mfa.manualKey')}
                </p>
                <code className="mt-1 block break-all text-sm font-semibold">
                  {setup.secret}
                </code>
              </div>
              <p className="text-xs text-slate-500">{t('mfa.qrWarning')}</p>
              <form
                className="max-w-md space-y-3"
                onSubmit={(event) => {
                  event.preventDefault()
                  if (/^\d{6}$/.test(code)) confirm.mutate(code)
                }}
              >
                <p className="text-sm font-semibold">{t('mfa.sixDigitCode')}</p>
                <MfaCodeInput
                  value={code}
                  onChange={setCode}
                  onComplete={(value) => confirm.mutate(value)}
                  disabled={confirm.isPending}
                />
                {confirm.isError ? (
                  <p role="alert" className="text-sm text-rose-700">
                    {errorMessage(confirm.error)}
                  </p>
                ) : null}
                <button
                  className="button-primary"
                  disabled={confirm.isPending || !/^\d{6}$/.test(code)}
                >
                  {t('mfa.enable')}
                </button>
              </form>
            </div>
          ) : null}
          {status.data.enabled && recoveryCodes.length === 0 ? (
            <form
              className="mt-5 max-w-md space-y-3"
              onSubmit={(event) => {
                event.preventDefault()
                if (
                  password &&
                  (factorMode === 'app'
                    ? /^\d{6}$/.test(code)
                    : /^[0-9a-fA-F]{32}$/.test(code))
                )
                  disable.mutate(code)
              }}
            >
              <h3 className="flex items-center gap-2 text-sm font-bold">
                <ShieldOff size={17} aria-hidden="true" />{' '}
                {t('mfa.disableTitle')}
              </h3>
              <p className="text-xs text-slate-500">{t('mfa.disableHint')}</p>
              <label className="field">
                {t('profile.currentPassword')}
                <input
                  className="form-control"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </label>
              <div className="space-y-3">
                <p className="text-sm font-semibold">
                  {t('auth.securityCode')}
                </p>
                {factorMode === 'app' ? (
                  <MfaCodeInput
                    value={code}
                    onChange={setCode}
                    onComplete={(value) => {
                      if (password) disable.mutate(value)
                    }}
                    disabled={disable.isPending}
                  />
                ) : (
                  <input
                    className="form-control w-full font-mono"
                    autoComplete="off"
                    maxLength={32}
                    placeholder={t('auth.recoveryCode')}
                    value={code}
                    onChange={(event) => setCode(event.target.value)}
                  />
                )}
                <button
                  type="button"
                  className="button-ghost"
                  onClick={() => {
                    setCode('')
                    setFactorMode((mode) =>
                      mode === 'app' ? 'recovery' : 'app',
                    )
                  }}
                >
                  {factorMode === 'app'
                    ? t('auth.useRecoveryCode')
                    : t('auth.useAuthenticator')}
                </button>
              </div>
              {disable.isError ? (
                <p role="alert" className="text-sm text-rose-700">
                  {errorMessage(disable.error)}
                </p>
              ) : null}
              <button
                className="button-danger"
                disabled={
                  disable.isPending ||
                  !password ||
                  (factorMode === 'app'
                    ? !/^\d{6}$/.test(code)
                    : !/^[0-9a-fA-F]{32}$/.test(code))
                }
              >
                {t('mfa.disable')}
              </button>
            </form>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
