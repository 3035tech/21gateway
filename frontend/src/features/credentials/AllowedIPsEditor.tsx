import { useState } from 'react'
import { Plus, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

type EntryKind = 'address' | 'network'

function normalizeEntry(
  raw: string,
): { value: string; kind: EntryKind } | null {
  const parts = raw.trim().split('/')
  if (parts.length < 1 || parts.length > 2 || !parts[0]) return null
  const address = parts[0]
  let normalizedAddress: string
  let maximumPrefix: number
  if (address.includes(':')) {
    if (address.includes('%')) return null
    try {
      normalizedAddress = new URL(`http://[${address}]/`).hostname.slice(1, -1)
    } catch {
      return null
    }
    if (normalizedAddress.startsWith('::ffff:')) return null
    maximumPrefix = 128
  } else {
    const octets = address.split('.')
    if (
      octets.length !== 4 ||
      octets.some(
        (octet) => !/^(0|[1-9][0-9]{0,2})$/.test(octet) || Number(octet) > 255,
      )
    )
      return null
    normalizedAddress = octets.join('.')
    maximumPrefix = 32
  }
  if (parts.length === 1) return { value: normalizedAddress, kind: 'address' }
  if (!/^(0|[1-9][0-9]{0,2})$/.test(parts[1])) return null
  const prefix = Number(parts[1])
  if (prefix > maximumPrefix) return null
  return { value: `${normalizedAddress}/${prefix}`, kind: 'network' }
}

export function AllowedIPsEditor({
  value,
  onChange,
  onValidityChange,
  disabled = false,
}: {
  value: string
  onChange: (value: string) => void
  onValidityChange: (valid: boolean) => void
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState('')
  const [error, setError] = useState('')
  const entries = value.split(/[\s,;]+/).filter(Boolean)
  const addresses = entries.filter((entry) => !entry.includes('/'))
  const networks = entries.filter((entry) => entry.includes('/'))

  function addDraft() {
    const candidates = draft.split(/[\s,;]+/).filter(Boolean)
    if (!candidates.length) return
    const next = [...entries]
    for (const candidate of candidates) {
      const normalized = normalizeEntry(candidate)
      if (!normalized) {
        setError(t('credentials.invalidIpEntry', { value: candidate }))
        onValidityChange(false)
        return
      }
      if (!next.includes(normalized.value)) next.push(normalized.value)
    }
    if (next.length > 100) {
      setError(t('credentials.ipLimit'))
      onValidityChange(false)
      return
    }
    onChange(next.join('\n'))
    setDraft('')
    setError('')
    onValidityChange(true)
  }

  return (
    <div className="space-y-4">
      <div>
        <label htmlFor="allowed-ip-entry" className="field">
          {t('credentials.addIpEntry')}
        </label>
        <div className="mt-2 flex flex-col gap-2 sm:flex-row">
          <input
            id="allowed-ip-entry"
            className="form-control min-w-0 flex-1 font-mono"
            autoComplete="off"
            spellCheck={false}
            value={draft}
            onChange={(event) => {
              setDraft(event.target.value)
              setError('')
              onValidityChange(true)
            }}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault()
                addDraft()
              }
            }}
            onBlur={addDraft}
            disabled={disabled}
            placeholder="203.0.113.10 ou 198.51.100.0/24"
            aria-invalid={!!error}
          />
          <button
            type="button"
            className="button-secondary shrink-0"
            onMouseDown={(event) => event.preventDefault()}
            onClick={addDraft}
            disabled={disabled || !draft.trim()}
          >
            <Plus size={16} aria-hidden="true" /> {t('credentials.addIp')}
          </button>
        </div>
        <p className="mt-2 text-xs leading-5 text-slate-500">
          {t('credentials.ipHint')}
        </p>
        {error ? (
          <p role="alert" className="mt-2 text-xs text-rose-700">
            {error}
          </p>
        ) : null}
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        {(
          [
            ['address', addresses, 'credentials.singleIps'],
            ['network', networks, 'credentials.ipNetworks'],
          ] as const
        ).map(([kind, group, title]) => (
          <section
            key={kind}
            className="min-w-0 rounded-xl border border-slate-200 bg-slate-50/60 p-4"
            aria-label={t(title)}
          >
            <h4 className="text-sm font-bold text-slate-900">
              {t(title)}{' '}
              <span className="font-normal text-slate-500">
                ({group.length})
              </span>
            </h4>
            {group.length ? (
              <ul className="mt-3 flex flex-wrap gap-2">
                {group.map((entry) => (
                  <li
                    key={entry}
                    className="inline-flex max-w-full items-center gap-1 rounded-lg border border-purple-100 bg-white pl-3 pr-1 text-sm text-slate-800"
                  >
                    <span className="break-all font-mono">{entry}</span>
                    <button
                      type="button"
                      className="button-ghost min-h-9 min-w-9 p-1"
                      aria-label={t('credentials.removeIp', { value: entry })}
                      onClick={() =>
                        onChange(
                          entries.filter((value) => value !== entry).join('\n'),
                        )
                      }
                      disabled={disabled}
                    >
                      <X size={14} aria-hidden="true" />
                    </button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="mt-3 text-xs text-slate-500">
                {t('credentials.noIpEntries')}
              </p>
            )}
          </section>
        ))}
      </div>
    </div>
  )
}
