package asyncapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amer8/apibconv/pkg/model"
)

// effectiveContentType applies the document default only when the message
// does not specify its own type. Keep the historical JSON fallback when
// neither source provides a type.
func effectiveContentType(messageType, documentDefault string) string {
	if messageType != "" {
		return messageType
	}
	if documentDefault != "" {
		return documentDefault
	}
	return "application/json"
}

// messageSchema resolves message references before interpreting their payload as a schema.
func (p *Parser) messageSchema(raw any, components map[string]interface{}, channels map[string]Channel3, v3 bool) (*model.Schema, string, error) {
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
		if message.Headers != nil {
			return nil, "", fmt.Errorf("message headers cannot be represented without loss")
		}
		if message.SchemaFormat != nil {
			return nil, "", fmt.Errorf("explicit message schemaFormat %q cannot be represented without loss", *message.SchemaFormat)
		}
		if message.Ref == "" {
			contentType, err := messageTraitContentType(&message, components, v3)
			if err != nil {
				return nil, "", err
			}
			if message.Payload == nil {
				return nil, contentType, nil
			}
			schema := p.convertSchemaPayload(message.Payload)
			if schema == nil {
				return nil, "", fmt.Errorf("unsupported message payload")
			}
			return schema, contentType, nil
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

// V2 uses JSON merge patch in trait order, while V3 forbids traits from
// overriding a property explicitly set on the target message.
func messageTraitContentType(message *Message, components map[string]interface{}, v3 bool) (string, error) {
	contentType := message.ContentType
	explicit := contentType != ""
	for _, trait := range message.Traits {
		seen := make(map[string]bool)
		for {
			rawRef, hasRef := trait["$ref"]
			if !hasRef {
				break
			}
			ref, ok := rawRef.(string)
			const prefix = "#/components/messageTraits/"
			if !ok || !strings.HasPrefix(ref, prefix) {
				return "", fmt.Errorf("unsupported message trait reference %q", rawRef)
			}
			name, err := decodePointerToken(strings.TrimPrefix(ref, prefix))
			if err != nil {
				return "", err
			}
			if seen[name] {
				return "", fmt.Errorf("cyclic message trait reference %q", ref)
			}
			seen[name] = true
			traits, _ := components["messageTraits"].(map[string]interface{})
			raw, ok := traits[name]
			if !ok {
				return "", fmt.Errorf("message trait reference %q not found", ref)
			}
			data, err := json.Marshal(raw)
			if err != nil {
				return "", fmt.Errorf("encode message trait %q: %w", ref, err)
			}
			trait = nil
			if err := json.Unmarshal(data, &trait); err != nil || trait == nil {
				return "", fmt.Errorf("invalid message trait %q", ref)
			}
		}
		if trait["headers"] != nil {
			return "", fmt.Errorf("message trait headers cannot be represented without loss")
		}
		if trait["schemaFormat"] != nil {
			return "", fmt.Errorf("message trait schemaFormat cannot be represented without loss")
		}
		raw, ok := trait["contentType"]
		if !ok || (v3 && explicit) {
			continue
		}
		if raw == nil {
			contentType = ""
			continue
		}
		value, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("message trait contentType must be a string")
		}
		contentType = value
		if v3 {
			explicit = true
		}
	}
	return contentType, nil
}
