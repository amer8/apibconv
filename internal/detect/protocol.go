package detect

import (
	"strings"
)

// Protocol attempts to guess the AsyncAPI protocol from a server URL.
func Protocol(url string) string {
	lowerURL := strings.ToLower(url)

	switch {
	case strings.HasPrefix(lowerURL, "amqps://"):
		return "amqps"
	case strings.HasPrefix(lowerURL, "amqp://"):
		return "amqp"
	case strings.HasPrefix(lowerURL, "mqtts://"):
		return "mqtts"
	case strings.HasPrefix(lowerURL, "mqtt://"):
		return "mqtt"
	case strings.HasPrefix(lowerURL, "wss://"):
		return "wss"
	case strings.HasPrefix(lowerURL, "ws://"):
		return "ws"
	case strings.HasPrefix(lowerURL, "kafka://"): // Not standard standard URI but common convention
		return "kafka"
	case strings.HasPrefix(lowerURL, "https://"):
		return "https"
	case strings.HasPrefix(lowerURL, "http://"):
		return "http"
	case strings.Contains(lowerURL, "kafka"):
		return "kafka"
	case strings.Contains(lowerURL, "rabbitmq"):
		return "amqp"
	case strings.Contains(lowerURL, "mosquitto"), strings.Contains(lowerURL, "hivemq"): //nolint:misspell // proper noun
		return "mqtt"
	default:
		return "unknown"
	}
}
