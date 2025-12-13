package detect

import (
	"strings"
)

// Protocol attempts to guess the AsyncAPI protocol from a server URL.
func Protocol(url string) string {
	lowerURL := strings.ToLower(url)

	switch {
	case strings.HasPrefix(lowerURL, "amqp://"), strings.HasPrefix(lowerURL, "amqps://"):
		return "amqp"
	case strings.HasPrefix(lowerURL, "mqtt://"), strings.HasPrefix(lowerURL, "mqtts://"):
		return "mqtt"
	case strings.HasPrefix(lowerURL, "ws://"), strings.HasPrefix(lowerURL, "wss://"):
		return "ws"
	case strings.HasPrefix(lowerURL, "kafka://"): // Not standard standard URI but common convention
		return "kafka"
	case strings.HasPrefix(lowerURL, "http://"), strings.HasPrefix(lowerURL, "https://"):
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
