import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { ClipboardCheck, Send, ShieldCheck, Webhook } from 'lucide-react'
import { Dialog } from '../../components/admin/Dialog'
import { errorMessage } from '../../lib/admin'
import { api } from '../../lib/api'
import { listMessages, sendTestSMS } from '../messages/api'
import type { Credential } from './api'

export function CredentialTestDialog({
  credential,
  onClose,
}: {
  credential: Credential
  onClose: () => void
}) {
  const [destination, setDestination] = useState('')
  const [confirmed, setConfirmed] = useState(false)
  const [delivery, setDelivery] = useState<string | null>(null)
  const checkURL = new URL(
    '/v1/auth/check',
    api.defaults.baseURL ?? window.location.origin,
  ).toString()
  const send = useMutation({
    mutationFn: () => sendTestSMS(credential.id, destination),
    onSuccess: () => setDelivery(null),
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
      title={`Testar credencial · ${credential.username}`}
      onClose={onClose}
      busy={busy}
      wide
    >
      <div className="space-y-6">
        <section className="rounded-xl border border-slate-200 p-5">
          <h3 className="flex items-center gap-2 font-bold">
            <ShieldCheck size={18} className="text-brand-purple" /> 1. Acesso do
            cliente
          </h3>
          <p className="mt-3 text-sm leading-6 text-slate-600">
            Peça ao cliente para chamar esta rota a partir da própria rede,
            usando o usuário <strong>{credential.username}</strong> e a senha da
            API. Resposta 204 confirma senha e IP; 401 indica falha de
            autenticação e 403 indica IP não autorizado.
          </p>
          <code className="mt-3 block overflow-x-auto rounded-lg bg-slate-100 p-3 text-xs text-slate-800">
            GET {checkURL}
          </code>
          <p className="mt-2 text-xs text-slate-500">
            Esta checagem não envia SMS nem consulta status. A whitelist usa o IP
            visto pelo gateway; confirme a configuração de proxy/NAT no ambiente.
          </p>
        </section>
        <section className="rounded-xl border border-slate-200 p-5">
          <h3 className="flex items-center gap-2 font-bold">
            <Send size={18} className="text-brand-purple" /> 2. Aceite pelo
            white label
          </h3>
          <p className="mt-3 text-sm leading-6 text-slate-600">
            Envia uma mensagem fixa pelo caminho normal desta credencial. O
            número deve ser o destino de homologação autorizado na configuração
            do servidor.
          </p>
          <label className="mt-4 block text-sm font-semibold">
            Número de teste (DDD + número)
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
            Confirmo que este número foi autorizado e que o teste pode gerar
            cobrança de um SMS.
          </label>
          {send.isError ? (
            <p role="alert" className="mt-3 text-sm text-rose-700">
              {errorMessage(send.error)} Se a chamada ficou sem resposta,
              consulte Envios antes de tentar novamente.
            </p>
          ) : null}
          {send.data ? (
            <div
              role="status"
              className={`mt-4 rounded-lg p-4 text-sm ${send.data.submissionState === 'accepted' ? 'bg-emerald-50 text-emerald-800' : 'bg-amber-50 text-amber-900'}`}
            >
              <p className="font-bold">
                {send.data.submissionState === 'accepted'
                  ? 'Fornecedor aceitou o SMS'
                  : send.data.submissionState === 'unknown'
                    ? 'Resultado incerto: não reenvie automaticamente'
                    : 'Fornecedor não aceitou o SMS'}
              </p>
              {send.data.idSms ? (
                <p className="mt-1">ID do envio: {send.data.idSms}</p>
              ) : null}
              <p className="mt-1 break-all">
                Referência: {send.data.correlationId}
              </p>
              {send.data.failureCode ? (
                <p className="mt-1">Código: {send.data.failureCode}</p>
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
              Enviar um SMS de teste
            </button>
          ) : null}
        </section>
        <section className="rounded-xl border border-slate-200 p-5">
          <h3 className="flex items-center gap-2 font-bold">
            <Webhook size={18} className="text-brand-purple" /> 3. Entrega ao
            destinatário
          </h3>
          <p className="mt-3 text-sm leading-6 text-slate-600">
            O aceite acima não confirma a entrega. Quando o DLR estiver
            homologado e chegar ao gateway, consulte o estado registrado.
          </p>
          {send.data?.idSms ? (
            <button
              type="button"
              className="button-secondary mt-4"
              disabled={busy}
              onClick={() => checkDelivery.mutate()}
            >
              <ClipboardCheck size={16} /> Verificar DLR
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
                ? `Status recebido: ${delivery}`
                : 'Sem confirmação de entrega registrada.'}
            </p>
          ) : null}
        </section>
      </div>
    </Dialog>
  )
}
