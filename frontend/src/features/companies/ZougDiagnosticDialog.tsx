import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Dialog } from '../../components/admin/Dialog'
import { errorMessage } from '../../lib/admin'
import { runZougDiagnostic } from './api'

export function ZougDiagnosticDialog({
  publicRef,
  onClose,
}: {
  publicRef: string
  onClose: () => void
}) {
  const { t } = useTranslation()
  const [includeDestination, setIncludeDestination] = useState(false)
  const [destination, setDestination] = useState('')
  const diagnostic = useMutation({
    mutationFn: () =>
      runZougDiagnostic(publicRef, includeDestination ? destination : ''),
  })
  return (
    <Dialog
      title={t('companyDetail.diagnosticTitle')}
      onClose={onClose}
      busy={diagnostic.isPending}
      wide
    >
      <p className="mb-4 text-sm text-slate-600">
        {t('companyDetail.diagnosticHint')}
      </p>
      <label className="mb-3 flex items-center gap-2 text-sm font-semibold text-slate-700">
        <input
          type="checkbox"
          checked={includeDestination}
          onChange={(event) => setIncludeDestination(event.target.checked)}
          disabled={diagnostic.isPending}
        />
        {t('companyDetail.diagnosticIncludeDestination')}
      </label>
      {includeDestination ? (
        <label className="mb-4 block text-sm font-semibold text-slate-700">
          {t('companyDetail.diagnosticDestination')}
          <input
            className="form-control mt-2 w-full"
            type="text"
            inputMode="numeric"
            autoComplete="off"
            value={destination}
            onChange={(event) =>
              setDestination(event.target.value.replace(/\D/g, '').slice(0, 11))
            }
            disabled={diagnostic.isPending}
          />
        </label>
      ) : null}
      <button
        className="button-primary"
        disabled={
          diagnostic.isPending ||
          (includeDestination && destination.length !== 11)
        }
        onClick={() => diagnostic.mutate()}
      >
        {diagnostic.isPending
          ? t('companyDetail.diagnosticRunning')
          : t('companyDetail.diagnosticRun')}
      </button>
      {diagnostic.isError ? (
        <p role="alert" className="mt-4 text-sm text-rose-700">
          {errorMessage(diagnostic.error)}
        </p>
      ) : null}
      {diagnostic.data ? (
        <div className="mt-5 space-y-3">
          <p className="break-all text-xs text-slate-500">
            Run ID: {diagnostic.data.runId}
          </p>
          {diagnostic.data.results.map((result) => (
            <details
              key={result.caseId}
              className="rounded-xl border border-slate-200 p-3"
            >
              <summary className="cursor-pointer font-medium text-slate-900">
                {result.caseId} · HTTP {result.httpStatus || result.errorCode} ·{' '}
                {result.durationMs} ms
              </summary>
              <dl className="mt-3 space-y-2 text-xs">
                <div>
                  <dt className="font-semibold">
                    {t('companyDetail.diagnosticRequest')}
                  </dt>
                  <dd className="mt-1 break-all">
                    {result.requestMethod} {result.requestPath}; Content-Type:{' '}
                    {result.contentType}; auth: {result.auth}
                  </dd>
                  <dd>
                    <pre className="mt-1 overflow-x-auto whitespace-pre-wrap break-all rounded bg-slate-50 p-2">
                      {result.requestBody}
                    </pre>
                  </dd>
                </div>
                <div>
                  <dt className="font-semibold">
                    {t('companyDetail.diagnosticResponse')}
                  </dt>
                  <dd>
                    <pre className="mt-1 overflow-x-auto whitespace-pre-wrap break-all rounded bg-slate-50 p-2">
                      {JSON.stringify(result.responseHeaders ?? {}, null, 2)}
                    </pre>
                  </dd>
                  <dd>
                    <pre className="mt-1 overflow-x-auto whitespace-pre-wrap break-all rounded bg-slate-50 p-2">
                      {result.responseBody || result.errorCode || '—'}
                    </pre>
                  </dd>
                </div>
              </dl>
            </details>
          ))}
        </div>
      ) : null}
    </Dialog>
  )
}
