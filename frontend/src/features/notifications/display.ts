import type { TFunction } from 'i18next'

const incidentDescriptions: Record<string, string> = {
  unknown_sends: 'envios sem resultado conclusivo; investigue antes de reenviar.',
  expired_leases: 'jobs com lease expirado; confira os workers.',
  dlr_pending: 'notificações DLR pendentes há mais de 15 minutos.',
  mo_pending: 'notificações MO pendentes há mais de 15 minutos.',
  dlr_unmatched: 'eventos DLR sem envio associado há mais de 15 minutos.',
  mo_unmatched: 'eventos MO sem envio associado há mais de 15 minutos.',
  schedule_overdue: 'SMS agendados atrasados há mais de 5 minutos.',
  callback_failed: 'notificações ao cliente com falha definitiva na última hora.',
}

export function notificationTitle(t: TFunction, code: string, fallback: string) {
  return Object.hasOwn(incidentDescriptions, code)
    ? t(`notifications.types.${code}`)
    : fallback
}

export function notificationDescription(
  t: TFunction,
  code: string,
  fallback: string,
) {
  const match = /^(\d+) (.+)$/.exec(fallback)
  if (!match || incidentDescriptions[code] !== match[2]) return fallback
  const count = Number(match[1])
  return Number.isSafeInteger(count)
    ? t(`notifications.details.${code}`, { count })
    : fallback
}
