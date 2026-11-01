package models

// Provider identifies an external messaging integration, independently of channel.
type Provider string

const (
	ProviderZoug   Provider = "zoug"
	ProviderPontal Provider = "pontal"
	ProviderMKM    Provider = "mkm"
	ProviderZenvia Provider = "zenvia"
	ProviderGoogle Provider = "google"
)

// Channel identifies the product carried by a provider. Only SMS is active today.
type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelRCS      Channel = "rcs"
	ChannelWhatsApp Channel = "whatsapp"
)

type ProviderRouteStatus string

const (
	ProviderRouteActive  ProviderRouteStatus = "active"
	ProviderRoutePlanned ProviderRouteStatus = "planned"
)

type CompanyProviderRoute struct {
	CompanyID    uint64              `gorm:"column:company_id"`
	Channel      Channel             `gorm:"column:channel"`
	ProviderCode Provider            `gorm:"column:provider_code"`
	Status       ProviderRouteStatus `gorm:"column:status"`
}

func (CompanyProviderRoute) TableName() string { return "company_provider_routes" }

func (p Provider) PlannedFor(channel Channel) bool {
	switch channel {
	case ChannelRCS:
		return p == ProviderGoogle || p == ProviderZenvia
	case ChannelWhatsApp:
		return p == ProviderZenvia
	default:
		return false
	}
}

func (p Provider) AvailableForSMS() bool { return p == ProviderZoug }
