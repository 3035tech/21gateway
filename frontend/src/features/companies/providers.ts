export const SMS_PROVIDER_ZOUG = 'zoug' as const
export const SMS_PROVIDER_PONTAL = 'pontal' as const
export const SMS_PROVIDER_MKM = 'mkm' as const
export const SMS_PROVIDER_ZENVIA = 'zenvia' as const
export const PROVIDER_GOOGLE = 'google' as const
export const availableSMSProviders = [SMS_PROVIDER_ZOUG] as const

export const CHANNEL_SMS = 'sms' as const
export const CHANNEL_RCS = 'rcs' as const
export const CHANNEL_WHATSAPP = 'whatsapp' as const
export const ROUTE_ACTIVE = 'active' as const
export const ROUTE_PLANNED = 'planned' as const

export type ProviderCode =
  | typeof SMS_PROVIDER_ZOUG
  | typeof SMS_PROVIDER_PONTAL
  | typeof SMS_PROVIDER_MKM
  | typeof SMS_PROVIDER_ZENVIA
  | typeof PROVIDER_GOOGLE
export type ChannelCode =
  typeof CHANNEL_SMS | typeof CHANNEL_RCS | typeof CHANNEL_WHATSAPP

// Catalog capabilities are informational until a matching adapter is active.
export const providerChannels: Record<ProviderCode, readonly ChannelCode[]> = {
  [SMS_PROVIDER_ZOUG]: [CHANNEL_SMS],
  [SMS_PROVIDER_PONTAL]: [CHANNEL_SMS],
  [SMS_PROVIDER_MKM]: [CHANNEL_SMS],
  [SMS_PROVIDER_ZENVIA]: [CHANNEL_SMS, CHANNEL_RCS, CHANNEL_WHATSAPP],
  [PROVIDER_GOOGLE]: [CHANNEL_RCS],
}

export const channelLabels: Record<ChannelCode, string> = {
  [CHANNEL_SMS]: 'SMS',
  [CHANNEL_RCS]: 'RCS',
  [CHANNEL_WHATSAPP]: 'WhatsApp',
}

export const providerOptions = [
  { value: SMS_PROVIDER_ZOUG, label: 'Zoug' },
  { value: SMS_PROVIDER_PONTAL, label: 'Pontal' },
  { value: SMS_PROVIDER_MKM, label: 'MKM' },
  { value: SMS_PROVIDER_ZENVIA, label: 'Zenvia' },
  { value: PROVIDER_GOOGLE, label: 'Google' },
]

export function providerLabel(code?: string) {
  return (
    providerOptions.find((provider) => provider.value === code)?.label ??
    code ??
    ''
  )
}

export function canSaveProviderChannel(
  provider: ProviderCode,
  channel: ChannelCode,
) {
  return provider === SMS_PROVIDER_ZOUG && channel === CHANNEL_SMS
}

export function canPlanProviderChannel(
  provider: ProviderCode,
  channel: ChannelCode,
) {
  return (
    (channel === CHANNEL_RCS &&
      (provider === PROVIDER_GOOGLE || provider === SMS_PROVIDER_ZENVIA)) ||
    (channel === CHANNEL_WHATSAPP && provider === SMS_PROVIDER_ZENVIA)
  )
}
