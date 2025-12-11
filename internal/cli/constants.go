package cli

const (
	// Formats supported by the CLI.
	formatOpenAPI  = "openapi"
	formatAsyncAPI = "asyncapi"
	formatAPIB     = "apib"

	// Encodings supported by the CLI.
	encodingJSON = "json"
	encodingYAML = "yaml"
	// encodingYML is used for file extension checks.
	encodingYML = "yml"

	// Defaults for flags.
	defaultOpenAPIVersion  = "3.0"
	defaultAsyncAPIVersion = "2.6"
)
