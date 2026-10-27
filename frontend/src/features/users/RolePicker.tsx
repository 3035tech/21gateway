import { Eye, Settings2, ShieldCheck } from 'lucide-react'
import { roleDescriptions, roleLabels } from '../../lib/admin'
import type { UserBody } from './api'

type Role = UserBody['role']

const options = [
  { value: 'viewer', Icon: Eye },
  { value: 'operator', Icon: Settings2 },
  { value: 'admin', Icon: ShieldCheck },
] as const

export function RolePicker({
  value,
  onChange,
  disabled = false,
}: {
  value: Role
  onChange: (value: Role) => void
  disabled?: boolean
}) {
  return (
    <fieldset className="space-y-2" disabled={disabled}>
      <legend className="mb-2 text-sm font-semibold text-slate-900">
        Perfil de acesso
      </legend>
      {options.map(({ value: role, Icon }) => {
        const selected = role === value
        return (
          <label
            key={role}
            className={`flex cursor-pointer items-start gap-3 rounded-xl border px-4 py-3.5 transition-colors focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-brand-purple ${selected ? 'border-brand-purple bg-purple-50' : 'border-slate-200 bg-white hover:border-purple-300'} ${disabled ? 'cursor-not-allowed opacity-60' : ''}`}
          >
            <input
              type="radio"
              name="system-user-role"
              value={role}
              checked={selected}
              disabled={disabled}
              onChange={() => onChange(role)}
              className="sr-only"
            />
            <span
              className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${selected ? 'bg-brand-purple text-white' : 'bg-slate-100 text-slate-600'}`}
              aria-hidden="true"
            >
              <Icon size={18} />
            </span>
            <span className="min-w-0">
              <span className="block text-sm font-bold text-slate-900">
                {roleLabels[role]}
              </span>
              <span className="mt-0.5 block text-xs leading-5 text-slate-600">
                {roleDescriptions[role]}
              </span>
            </span>
            <span
              className={`ml-auto mt-2 h-4 w-4 shrink-0 rounded-full border-2 ${selected ? 'border-brand-purple bg-brand-purple shadow-[inset_0_0_0_3px_white]' : 'border-slate-300'}`}
              aria-hidden="true"
            />
          </label>
        )
      })}
    </fieldset>
  )
}
