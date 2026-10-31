export const SMS_PROVIDER_ZOUG = 'zoug' as const
export const SMS_PROVIDER_PONTAL = 'pontal' as const
export const SMS_PROVIDER_MKM = 'mkm' as const
export const SMS_PROVIDER_ZENVIA = 'zenvia' as const
export const availableSMSProviders = [SMS_PROVIDER_ZOUG] as const

export const CHANNEL_SMS = 'sms' as const
export const CHANNEL_RCS = 'rcs' as const
export const CHANNEL_WHATSAPP = 'whatsapp' as const

export type ProviderCode =
  | typeof SMS_PROVIDER_ZOUG
  | typeof SMS_PROVIDER_PONTAL
  | typeof SMS_PROVIDER_MKM
  | typeof SMS_PROVIDER_ZENVIA
export type ChannelCode =
  | typeof CHANNEL_SMS
  | typeof CHANNEL_RCS
  | typeof CHANNEL_WHATSAPP

// Catalog capabilities are informational until a matching adapter is active.
export const providerChannels: Record<ProviderCode, readonly ChannelCode[]> = {
  [SMS_PROVIDER_ZOUG]: [CHANNEL_SMS],
  [SMS_PROVIDER_PONTAL]: [CHANNEL_SMS],
  [SMS_PROVIDER_MKM]: [CHANNEL_SMS],
  [SMS_PROVIDER_ZENVIA]: [CHANNEL_SMS, CHANNEL_RCS, CHANNEL_WHATSAPP],
}

export const channelLabels: Record<ChannelCode, string> = {
  [CHANNEL_SMS]: 'SMS',
  [CHANNEL_RCS]: 'RCS',
  [CHANNEL_WHATSAPP]: 'WhatsApp',
}

export const smsProviderOptions = [
  { value: SMS_PROVIDER_ZOUG, label: 'Zoug' },
  { value: SMS_PROVIDER_PONTAL, label: 'Pontal' },
  { value: SMS_PROVIDER_MKM, label: 'MKM' },
  { value: SMS_PROVIDER_ZENVIA, label: 'Zenvia' },
]

export function canSaveProviderChannel(provider: ProviderCode, channel: ChannelCode) {
  return provider === SMS_PROVIDER_ZOUG && channel === CHANNEL_SMS
}
