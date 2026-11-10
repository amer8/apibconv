package asyncapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amer8/apibconv/pkg/model"
)

// messageSchema resolves message references before interpreting their payload as a schema.
func (p *Parser) messageSchema(raw any, components map[string]interface{}, channels map[string]Channel3) (*model.Schema, string, error) {
	seen := map[string]bool{}
	for raw != nil {
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, "", fmt.Errorf("encode message: %w", err)
		}
		var message Message
		if err = json.Unmarshal(data, &message); err != nil {
			return nil, "", fmt.Errorf("decode message: %w", err)
		}
		if len(message.OneOf) != 0 {
			return nil, "", fmt.Errorf("message oneOf alternatives cannot be represented without loss")
		}
		if message.Ref == "" {
			if message.Payload == nil {
				return nil, message.ContentType, nil
			}
			schema := p.convertSchemaPayload(message.Payload)
			if schema == nil {
				return nil, "", fmt.Errorf("unsupported message payload")
			}
			return schema, message.ContentType, nil
		}
		ref := message.Ref
		if seen[ref] {
			return nil, "", fmt.Errorf("cyclic message reference %q", ref)
		}
		seen[ref] = true
		switch {
		case strings.HasPrefix(ref, "#/components/messages/"):
			token, err := decodePointerToken(strings.TrimPrefix(ref, "#/components/messages/"))
			if err != nil {
				return nil, "", err
			}
			messages, _ := components["messages"].(map[string]interface{})
			var ok bool
			raw, ok = messages[token]
			if !ok {
				return nil, "", fmt.Errorf("message reference %q not found", ref)
			}
		case strings.HasPrefix(ref, "#/channels/"):
			parts := strings.Split(strings.TrimPrefix(ref, "#/channels/"), "/")
			if len(parts) != 3 || parts[1] != "messages" {
				return nil, "", fmt.Errorf("unsupported message reference %q", ref)
			}
			channelID, err := decodePointerToken(parts[0])
			if err != nil {
				return nil, "", err
			}
			messageID, err := decodePointerToken(parts[2])
			if err != nil {
				return nil, "", err
			}
			channel, ok := channels[channelID]
			if !ok {
				return nil, "", fmt.Errorf("message channel %q not found", channelID)
			}
			raw, ok = channel.Messages[messageID]
			if !ok {
				return nil, "", fmt.Errorf("message reference %q not found", ref)
			}
		default:
			return nil, "", fmt.Errorf("unsupported message reference %q", ref)
		}
	}
	return nil, "", nil
}
