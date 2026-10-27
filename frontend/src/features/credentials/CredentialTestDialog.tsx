import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { ClipboardCheck, Send, ShieldCheck, Webhook } from 'lucide-react'
import { Dialog } from '../../components/admin/Dialog'
import { errorMessage } from '../../lib/admin'
import { api } from '../../lib/api'
import { listMessages, sendTestSMS } from '../messages/api'
import type { Credential } from './api'
import { useTranslation } from 'react-i18next'
import { useFeedback } from '../../components/admin/feedbackContext'

export function CredentialTestDialog({
  credential,
  onClose,
}: {
  credential: Credential
  onClose: () => void
}) {
  const { t } = useTranslation()
  const feedback = useFeedback()
  const [destination, setDestination] = useState('')
  const [confirmed, setConfirmed] = useState(false)
  const [delivery, setDelivery] = useState<string | null>(null)
  const checkURL = new URL(
    '/v1/auth/check',
    api.defaults.baseURL ?? window.location.origin,
  ).toString()
  const send = useMutation({
    mutationFn: () => sendTestSMS(credential.id, destination),
    onSuccess: (result) => {
      setDelivery(null)
      if (result.submissionState === 'accepted')
        feedback.success(t('feedback.sendAccepted'))
      else if (result.submissionState === 'unknown')
        feedback.error(t('feedback.sendUncertain'))
      else feedback.error(t('feedback.sendRejected'))
    },
    onError: (error) =>
      feedback.error(
        errorMessage(error) + ' ' + t('credentialTest.uncertainHint'),
      ),
  })
  const checkDelivery = useMutation({
    mutationFn: async () => {
      if (!send.data?.correlationId) return null
      const page = await listMessages({
        page: 1,
        pageSize: 1,
        search: send.data.correlationId,
        sort: 'createdAt',
        direction: 'desc',
      })
      return (
        page.items.find(
          (item) => item.correlationId === send.data?.correlationId,
        )?.deliveryStatus ?? null
      )
    },
    onSuccess: setDelivery,
  })
  const busy = send.isPending || checkDelivery.isPending
  return (
    <Dialog
      title={t('credentialTest.title', { username: credential.username })}
      onClose={onClose}
      busy={busy}
      wide
    >
      <div className="space-y-6">
        <section className="rounded-xl border border-slate-200 p-5">
          <h3 className="flex items-center gap-2 font-bold">
            <ShieldCheck size={18} className="text-brand-purple" />{' '}
            {t('credentialTest.clientAccess')}
          </h3>
          <p className="mt-3 text-sm leading-6 text-slate-600">
            {t('credentialTest.clientInstructions', {
              username: credential.username,
            })}
          </p>
          <code className="mt-3 block overflow-x-auto rounded-lg bg-slate-100 p-3 text-xs text-slate-800">
            GET {checkURL}
          </code>
          <p className="mt-2 text-xs text-slate-500">
            {t('credentialTest.clientHint')}
          </p>
        </section>
        <section className="rounded-xl border border-slate-200 p-5">
          <h3 className="flex items-center gap-2 font-bold">
            <Send size={18} className="text-brand-purple" />{' '}
            {t('credentialTest.providerAcceptance')}
          </h3>
          <p className="mt-3 text-sm leading-6 text-slate-600">
            {t('credentialTest.providerHint')}
          </p>
          <label className="mt-4 block text-sm font-semibold">
            {t('credentialTest.testNumber')}
            <input
              className="form-control mt-2 w-full max-w-xs"
              inputMode="numeric"
              autoComplete="off"
              maxLength={11}
              value={destination}
              onChange={(event) => setDestination(event.target.value)}
              disabled={busy || !!send.data}
              placeholder="11999999999"
            />
          </label>
          <label className="mt-4 flex items-start gap-3 text-sm leading-6">
            <input
              type="checkbox"
              className="mt-1 size-4 accent-brand-purple"
              checked={confirmed}
              onChange={(event) => setConfirmed(event.target.checked)}
              disabled={busy || !!send.data}
            />
            {t('credentialTest.confirmCharge')}
          </label>
          {send.isError ? (
            <p role="alert" className="mt-3 text-sm text-rose-700">
              {errorMessage(send.error)} {t('credentialTest.uncertainHint')}
            </p>
          ) : null}
          {send.data ? (
            <div
              role="status"
              className={`mt-4 rounded-lg p-4 text-sm ${send.data.submissionState === 'accepted' ? 'bg-emerald-50 text-emerald-800' : 'bg-amber-50 text-amber-900'}`}
            >
              <p className="font-bold">
                {send.data.submissionState === 'accepted'
                  ? t('credentialTest.accepted')
                  : send.data.submissionState === 'unknown'
                    ? t('credentialTest.unknown')
                    : t('credentialTest.rejected')}
              </p>
              {send.data.idSms ? (
                <p className="mt-1">
                  {t('credentialTest.messageId', { id: send.data.idSms })}
                </p>
              ) : null}
              <p className="mt-1 break-all">
                {t('credentialTest.reference', { id: send.data.correlationId })}
              </p>
              {send.data.failureCode ? (
                <p className="mt-1">
                  {t('credentialTest.code', { code: send.data.failureCode })}
                </p>
              ) : null}
            </div>
          ) : !send.isError ? (
            <button
              type="button"
              className="button-primary mt-4"
              disabled={
                !confirmed ||
                !/^\d{11}$/.test(destination) ||
                busy ||
                !credential.active
              }
              onClick={() => send.mutate()}
            >
              {t('credentialTest.sendTest')}
            </button>
          ) : null}
        </section>
        <section className="rounded-xl border border-slate-200 p-5">
          <h3 className="flex items-center gap-2 font-bold">
            <Webhook size={18} className="text-brand-purple" />{' '}
            {t('credentialTest.delivery')}
          </h3>
          <p className="mt-3 text-sm leading-6 text-slate-600">
            {t('credentialTest.deliveryHint')}
          </p>
          {send.data?.idSms ? (
            <button
              type="button"
              className="button-secondary mt-4"
              disabled={busy}
              onClick={() => checkDelivery.mutate()}
            >
              <ClipboardCheck size={16} /> {t('credentialTest.checkDlr')}
            </button>
          ) : null}
          {checkDelivery.isError ? (
            <p role="alert" className="mt-3 text-sm text-rose-700">
              {errorMessage(checkDelivery.error)}
            </p>
          ) : null}
          {checkDelivery.isSuccess ? (
            <p role="status" className="mt-3 text-sm">
              {delivery
                ? t('credentialTest.statusReceived', { status: delivery })
                : t('credentialTest.noDelivery')}
            </p>
          ) : null}
        </section>
      </div>
    </Dialog>
  )
}
