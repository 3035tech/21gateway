package models

// Provider identifies an external messaging integration, independently of channel.
type Provider string

const (
	ProviderZoug   Provider = "zoug"
	ProviderPontal Provider = "pontal"
	ProviderMKM    Provider = "mkm"
	ProviderZenvia Provider = "zenvia"
)

// Channel identifies the product carried by a provider. Only SMS is active today.
type Channel string

const ChannelSMS Channel = "sms"

func (p Provider) AvailableForSMS() bool { return p == ProviderZoug }
