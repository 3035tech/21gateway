import type { LucideIcon } from 'lucide-react'
import { Link } from 'react-router-dom'

type BaseProps = {
  icon: LucideIcon
  label: string
  danger?: boolean
  disabled?: boolean
}

function classes(danger: boolean) {
  return danger ? 'table-icon-action-danger' : 'table-icon-action'
}

export function TableIconButton({
  icon: Icon,
  label,
  danger = false,
  disabled = false,
  onClick,
}: BaseProps & { onClick: () => void }) {
  return (
    <button
      type="button"
      className={classes(danger)}
      aria-label={label}
      title={label}
      disabled={disabled}
      onClick={onClick}
    >
      <Icon size={18} strokeWidth={1.9} aria-hidden="true" />
    </button>
  )
}

export function TableIconLink({
  icon: Icon,
  label,
  danger = false,
  to,
  onClick,
}: BaseProps & { to: string; onClick?: () => void }) {
  return (
    <Link
      className={classes(danger)}
      aria-label={label}
      title={label}
      to={to}
      onClick={onClick}
    >
      <Icon size={18} strokeWidth={1.9} aria-hidden="true" />
    </Link>
  )
}
