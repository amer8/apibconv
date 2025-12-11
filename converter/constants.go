package converter

// APIBlueprintFormat is the standard format header for API Blueprint 1A.
const APIBlueprintFormat = "FORMAT: 1A"

// MediaTypeJSON is the MIME type for JSON.
const MediaTypeJSON = "application/json"

// AsyncAPI Versions
const (
	// AsyncAPIVersion26 is the version string for AsyncAPI 2.6.0.
	AsyncAPIVersion26 = "2.6.0"

	// AsyncAPIVersion30 is the version string for AsyncAPI 3.0.0.
	AsyncAPIVersion30 = "3.0.0"
)

// AsyncAPI Actions
const (
	ActionSend      = "send"
	ActionReceive   = "receive"
	ActionPublish   = "publish"
	ActionSubscribe = "subscribe"
)

// JSON Schema Types
const (
	TypeString  = "string"
	TypeNumber  = "number"
	TypeInteger = "integer"
	TypeBoolean = "boolean"
	TypeObject  = "object"
	TypeArray   = "array"
	TypeNull    = "null"
)

// Protocol represents the protocol used in AsyncAPI servers.
type Protocol string

const (
	// ProtocolKafka indicates the Kafka protocol.
	ProtocolKafka Protocol = "kafka"
	// ProtocolMQTT indicates the MQTT protocol.
	ProtocolMQTT  Protocol = "mqtt"
	// ProtocolAMQP indicates the AMQP protocol.
	ProtocolAMQP  Protocol = "amqp"
	// ProtocolWS indicates the WebSocket protocol.
	ProtocolWS    Protocol = "ws"
	// ProtocolHTTP indicates the HTTP protocol.
	ProtocolHTTP  Protocol = "http"
	// ProtocolHTTPS indicates the HTTPS protocol.
	ProtocolHTTPS Protocol = "https"
)
