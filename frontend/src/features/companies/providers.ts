export const SMS_PROVIDER_ZOUG = 'zoug' as const
export const SMS_PROVIDER_PONTAL = 'pontal' as const
export const SMS_PROVIDER_MKM = 'mkm' as const
export const SMS_PROVIDER_ZENVIA = 'zenvia' as const
export const availableSMSProviders = [SMS_PROVIDER_ZOUG] as const

// Planned providers remain visible in the company form but cannot be saved.
export const smsProviderOptions = [
  { value: SMS_PROVIDER_ZOUG, label: 'Zoug' },
  { value: SMS_PROVIDER_PONTAL, label: 'Pontal', disabled: true },
  { value: SMS_PROVIDER_MKM, label: 'MKM', disabled: true },
  { value: SMS_PROVIDER_ZENVIA, label: 'Zenvia', disabled: true },
]
