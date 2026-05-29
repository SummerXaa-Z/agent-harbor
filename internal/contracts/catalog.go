package contracts

import "github.com/SummerXaa-Z/agent-harbor/internal/domain"

const (
	providerSchemaVersion = "agent-provider-contract-v1"
	channelSchemaVersion  = "agent-channel-contract-v1"
)

func Providers() []domain.ProviderContract {
	return []domain.ProviderContract{
		{
			SchemaVersion:   providerSchemaVersion,
			Key:             "dify",
			Label:           "Dify",
			ChannelType:     "a2a",
			DefaultEndpoint: "https://api.example.com/dify/a2a",
			ChannelConfigFields: []domain.FieldContract{
				{Key: "provider", Type: "string", Required: true},
				{Key: "endpoint", Type: "url", Required: true, OutboundURL: true},
				{Key: "metadata", Type: "object", Required: false},
			},
			RequiredCreds:        []string{"apiBase", "apiKey"},
			OptionalCreds:        []string{"userId"},
			FutureMetadataPolicy: "metadata object only",
		},
		{
			SchemaVersion:   providerSchemaVersion,
			Key:             "webhook",
			Label:           "Webhook",
			ChannelType:     "webhook",
			DefaultEndpoint: "https://api.example.com/webhook",
			ChannelConfigFields: []domain.FieldContract{
				{Key: "provider", Type: "string", Required: true},
				{Key: "endpoint", Type: "url", Required: false, OutboundURL: true},
				{Key: "metadata", Type: "object", Required: false},
			},
			FutureMetadataPolicy: "metadata object only",
		},
	}
}

func Channels() []domain.ChannelContract {
	return []domain.ChannelContract{
		{
			SchemaVersion:              channelSchemaVersion,
			Key:                        "local",
			Label:                      "Local Caller",
			EndpointRequiredWhenActive: false,
			ChannelConfigFields: []domain.FieldContract{
				{Key: "description", Type: "string", Required: false},
				{Key: "metadata", Type: "object", Required: false},
			},
			FutureMetadataPolicy: "metadata object only",
		},
		{
			SchemaVersion:              channelSchemaVersion,
			Key:                        "mcp",
			Label:                      "MCP Server",
			EndpointRequiredWhenActive: true,
			ChannelConfigFields: []domain.FieldContract{
				{Key: "endpoint", Type: "url", Required: true, OutboundURL: true},
				{Key: "transport", Type: "string", Required: false},
				{Key: "metadata", Type: "object", Required: false},
			},
			FutureMetadataPolicy: "metadata object only",
		},
		{
			SchemaVersion:              channelSchemaVersion,
			Key:                        "openapi",
			Label:                      "OpenAPI Service",
			EndpointRequiredWhenActive: true,
			ChannelConfigFields: []domain.FieldContract{
				{Key: "endpoint", Type: "url", Required: true, OutboundURL: true},
				{Key: "specUrl", Type: "url", Required: false, OutboundURL: true},
				{Key: "metadata", Type: "object", Required: false},
			},
			FutureMetadataPolicy: "metadata object only",
		},
		{
			SchemaVersion:              channelSchemaVersion,
			Key:                        "a2a",
			Label:                      "A2A Agent",
			EndpointRequiredWhenActive: true,
			ChannelConfigFields: []domain.FieldContract{
				{Key: "endpoint", Type: "url", Required: true, OutboundURL: true},
				{Key: "provider", Type: "string", Required: false},
				{Key: "metadata", Type: "object", Required: false},
			},
			FutureMetadataPolicy: "metadata object only",
		},
		{
			SchemaVersion:              channelSchemaVersion,
			Key:                        "webhook",
			Label:                      "Webhook",
			EndpointRequiredWhenActive: false,
			ChannelConfigFields: []domain.FieldContract{
				{Key: "endpoint", Type: "url", Required: false, OutboundURL: true},
				{Key: "metadata", Type: "object", Required: false},
			},
			FutureMetadataPolicy: "metadata object only",
		},
	}
}

func KnownChannel(key string) bool {
	_, ok := Channel(key)
	return ok
}

func Channel(key string) (domain.ChannelContract, bool) {
	for _, channel := range Channels() {
		if channel.Key == key {
			return channel, true
		}
	}
	return domain.ChannelContract{}, false
}
