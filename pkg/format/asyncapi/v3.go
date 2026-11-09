package asyncapi

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/amer8/apibconv/internal/detect"
	"github.com/amer8/apibconv/pkg/model"
)

// parseV3 implements AsyncAPI 3.0 parsing
func (p *Parser) parseV3(data []byte) (*model.API, error) {
	var doc AsyncAPI3
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("asyncapi v3 parse failed: %w", err)
	}

	api := model.NewAPI()
	api.Version = doc.AsyncAPI
	api.Info.Title = doc.Info.Title
	api.Info.Version = doc.Info.Version
	api.Info.Description = doc.Info.Description
	security := newSecurityDecoder(api, doc.Components, true)

	// Servers (Sorted)
	serverNames := make([]string, 0, len(doc.Servers))
	for k := range doc.Servers {
		serverNames = append(serverNames, k)
	}
	sort.Strings(serverNames)

	var serverPolicies [][]model.SecurityRequirement
	policiesByServer := make(map[string][]model.SecurityRequirement)
	for _, name := range serverNames {
		s := doc.Servers[name]
		if s.Ref != "" {
			return nil, securityInputError("referenced server may contain security requirements")
		}
		policy, err := security.requirementsV3(s.Security)
		if err != nil {
			return nil, fmt.Errorf("server %q: %w", name, err)
		}
		serverPolicies = append(serverPolicies, policy)
		policiesByServer[name] = policy
		api.Servers = append(api.Servers, model.Server{
			Name:        name,
			URL:         s.Host,
			Protocol:    s.Protocol,
			Description: s.Description,
			Bindings:    s.Bindings,
			Security:    policy,
		})
	}
	var err error
	_, err = uniformServerSecurity(serverPolicies)
	if err != nil {
		return nil, err
	}

	// Map Channels by ID for lookup (needed for operations)
	channelMap := make(map[string]string) // ID -> Address
	for id, ch := range doc.Channels {
		if ch.Ref != "" {
			return nil, securityInputError("referenced channel may contain security requirements")
		}
		if ch.Address != "" {
			channelMap[id] = ch.Address
		} else {
			channelMap[id] = id // Fallback
		}
	}

	// Operations (Sorted)
	opIDs := make([]string, 0, len(doc.Operations))
	for k := range doc.Operations {
		opIDs = append(opIDs, k)
	}
	sort.Strings(opIDs)

	for _, opID := range opIDs {
		op := doc.Operations[opID]
		if op.Ref != "" {
			return nil, securityInputError("referenced operation may contain security requirements")
		}
		operationSecurity, err := security.operationSecurityV3(&op)
		if err != nil {
			return nil, fmt.Errorf("operation %q: %w", opID, err)
		}
		// Resolve channel address
		var path string
		var chRef string

		// Channel can be a reference or object.
		switch v := op.Channel.(type) {
		case string:
			chRef = v
		case map[string]interface{}:
			if ref, ok := v["$ref"].(string); ok {
				// extract ID from ref (e.g. #/channels/myChannel -> myChannel)
				if strings.HasPrefix(ref, "#/channels/") {
					chRef, err = decodePointerToken(strings.TrimPrefix(ref, "#/channels/"))
					if err != nil {
						return nil, fmt.Errorf("operation %q channel: %w", opID, err)
					}
				} else {
					if hasSecuredServer(policiesByServer) || hasActualRequirement(operationSecurity) {
						return nil, securityInputError(fmt.Sprintf("secured operation %q has an external channel reference", opID))
					}
					parts := strings.Split(ref, "/")
					if len(parts) > 0 {
						chRef = parts[len(parts)-1]
					}
				}
			}
		}

		if chRef == "" {
			if hasSecuredServer(policiesByServer) || hasActualRequirement(operationSecurity) {
				return nil, securityInputError(fmt.Sprintf("secured operation %q has no resolvable channel", opID))
			}
			continue
		}

		if addr, ok := channelMap[chRef]; ok {
			path = addr
		} else {
			path = chRef // Unknown ID, use as path
		}
		channel, foundChannel := doc.Channels[chRef]
		if !foundChannel && (hasActualRequirement(api.Security) || hasActualRequirement(operationSecurity) || len(policiesByServer) > 0) {
			return nil, securityInputError(fmt.Sprintf("operation %q references an unknown channel", opID))
		}
		serverNames, err := channelV3ServerNames(channel.Servers)
		if err != nil {
			return nil, fmt.Errorf("channel %q: %w", chRef, err)
		}
		serverSecurity, err := channelServerSecurity(serverNames, policiesByServer)
		if err != nil {
			return nil, fmt.Errorf("channel %q: %w", chRef, err)
		}

		// Get or Create PathItem
		pi, _ := api.GetPath(path)

		modelOp := &model.Operation{
			Summary:     op.Summary,
			Description: op.Description,
			Bindings:    op.Bindings,
			Security:    operationSecurity,
		}
		if modelOp.OperationID == "" {
			modelOp.OperationID = opID
		}

		switch op.Action {
		case "send":
			if pi.Post != nil && (hasActualRequirement(api.Security) || hasActualRequirement(serverSecurity) || hasActualRequirement(pi.Post.Security) || hasActualRequirement(modelOp.Security)) {
				return nil, securityInputError(fmt.Sprintf("multiple secured send operations share channel %q", path))
			}
			pi.Post = modelOp
		case "receive":
			if pi.Get != nil && (hasActualRequirement(api.Security) || hasActualRequirement(serverSecurity) || hasActualRequirement(pi.Get.Security) || hasActualRequirement(modelOp.Security)) {
				return nil, securityInputError(fmt.Sprintf("multiple secured receive operations share channel %q", path))
			}
			pi.Get = modelOp
		default:
			if hasActualRequirement(serverSecurity) || hasActualRequirement(operationSecurity) {
				return nil, securityInputError(fmt.Sprintf("secured operation %q has an unsupported action", opID))
			}
		}

		api.AddPath(path, &pi)
	}

	return api, nil
}

// writeV3 implements AsyncAPI 3.0 writing
func (w *Writer) writeV3(api *model.API, wr io.Writer, targetProtocol string, jsonOutput bool, plan *securityPlan) error {
	doc := AsyncAPI3{
		AsyncAPI:   w.version,
		Channels:   make(map[string]Channel3),
		Operations: make(map[string]Operation3),
		Servers:    make(map[string]ServerV3),
		Components: make(map[string]interface{}),
	}
	if len(plan.schemes) > 0 {
		doc.Components["securitySchemes"] = plan.schemes
	}

	doc.Info.Title = api.Info.Title
	doc.Info.Version = api.Info.Version
	doc.Info.Description = api.Info.Description

	// Servers
	for i, s := range api.Servers {
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("server%d", i)
		}
		var protocol string

		switch {
		case s.Protocol != "":
			protocol = s.Protocol
		case targetProtocol != "" && targetProtocol != "auto":
			protocol = targetProtocol
		case targetProtocol == "auto":
			protocol = detect.Protocol(s.URL)
		case s.Description != "" && len(s.Description) < 10: // Check if description is a short string, likely a protocol
			protocol = s.Description
		default:
			protocol = detect.Protocol(s.URL)
		}

		doc.Servers[name] = ServerV3{
			Host:        s.URL,
			Protocol:    protocol,
			Description: s.Description,
			Bindings:    s.Bindings,
			Security:    plan.serverV3[i],
		}
	}

	// Components -> Messages & Schemas
	messages := make(map[string]interface{})
	schemas := make(map[string]interface{})

	if len(api.Components.Schemas) > 0 {
		for name, schema := range api.Components.Schemas {
			data, _ := json.Marshal(schema)
			var schemaMap map[string]interface{}
			_ = json.Unmarshal(data, &schemaMap)
			schemas[name] = schemaMap

			messages[name] = Message{
				Name:        name,
				Title:       schema.Title,
				Summary:     schema.Description,
				ContentType: "application/json",
				Payload: map[string]interface{}{
					"$ref": "#/components/schemas/" + name,
				},
			}
		}
	}
	if len(schemas) > 0 {
		doc.Components["schemas"] = schemas
	}
	if len(messages) > 0 {
		doc.Components["messages"] = messages
	}

	// Paths -> Channels + Operations
	paths := make([]string, 0, len(api.Paths))
	for path := range api.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := api.Paths[path]

		// AsyncAPI channel addresses should not contain query params or fragments
		cleanPath := path
		if idx := strings.Index(path, "{?"); idx != -1 {
			cleanPath = path[:idx]
		}
		if idx := strings.Index(cleanPath, "#"); idx != -1 {
			cleanPath = cleanPath[:idx]
		}

		// Generate Channel ID
		channelID := item.Name
		if channelID == "" {
			channelID = cleanPath
		}

		// Escape Channel ID for JSON Pointer
		escapedChannelID := strings.ReplaceAll(channelID, "~", "~0")
		escapedChannelID = strings.ReplaceAll(escapedChannelID, "/", "~1")

		// Prepare Channel Messages Map
		channelMessages := make(map[string]interface{})

		if item.Post != nil {
			opID := item.Post.OperationID
			if opID == "" {
				opID = "send_" + channelID
			}
			op := Operation3{
				Action:      "send",
				Channel:     map[string]string{"$ref": "#/channels/" + escapedChannelID},
				Summary:     item.Post.Summary,
				Description: item.Post.Description,
				Bindings:    item.Post.Bindings,
				Security:    plan.v3[item.Post],
			}

			// Add Messages
			if item.Post.RequestBody != nil {
				for _, contentType := range sortedContentTypes(item.Post.RequestBody.Content) {
					mt := item.Post.RequestBody.Content[contentType]
					if mt.Schema == nil {
						continue
					}
					msg := &Message{}
					var msgKey string
					if mt.Schema.Ref != "" {
						ref := mt.Schema.Ref
						if strings.HasPrefix(ref, "#/components/schemas/") {
							name := strings.TrimPrefix(ref, "#/components/schemas/")
							msg.Ref = "#/components/messages/" + name
							msgKey = name
						} else {
							msg.Ref = ref
							parts := strings.Split(ref, "/")
							if len(parts) > 0 {
								msgKey = parts[len(parts)-1]
							} else {
								msgKey = asyncMessageKey("message", contentType)
							}
						}
					} else {
						data, _ := json.Marshal(mt.Schema)
						var payload interface{}
						_ = json.Unmarshal(data, &payload)
						msg.Payload = payload
						msgKey = asyncMessageKey("message", contentType)
					}

					// Add to channel
					channelMessages[msgKey] = msg

					// Add Ref to Operation
					// Message key also needs escaping if it contains special chars (unlikely for keys but good practice)
					escapedMsgKey := strings.ReplaceAll(msgKey, "~", "~0")
					escapedMsgKey = strings.ReplaceAll(escapedMsgKey, "/", "~1")

					opMsgRef := &Message{
						Ref: "#/channels/" + escapedChannelID + "/messages/" + escapedMsgKey,
					}
					op.Messages = append(op.Messages, opMsgRef)
					break
				}
			}
			doc.Operations[opID] = op
		}

		if item.Get != nil {
			opID := item.Get.OperationID
			if opID == "" {
				opID = "receive_" + channelID
			}
			op := Operation3{
				Action:      "receive",
				Channel:     map[string]string{"$ref": "#/channels/" + escapedChannelID},
				Summary:     item.Get.Summary,
				Description: item.Get.Description,
				Bindings:    item.Get.Bindings,
				Security:    plan.v3[item.Get],
			}

			// Add Messages
			if len(item.Get.Responses) > 0 {
				codes := make([]string, 0, len(item.Get.Responses))
				for code := range item.Get.Responses {
					codes = append(codes, code)
				}
				sort.Strings(codes)

				for _, code := range codes {
					resp := item.Get.Responses[code]
					for _, contentType := range sortedContentTypes(resp.Content) {
						mt := resp.Content[contentType]
						if mt.Schema == nil {
							continue
						}
						msg := &Message{}
						var msgKey string
						if mt.Schema.Ref != "" {
							ref := mt.Schema.Ref
							if strings.HasPrefix(ref, "#/components/schemas/") {
								name := strings.TrimPrefix(ref, "#/components/schemas/")
								msg.Ref = "#/components/messages/" + name
								msgKey = name
							} else {
								msg.Ref = ref
								parts := strings.Split(ref, "/")
								if len(parts) > 0 {
									msgKey = parts[len(parts)-1]
								} else {
									msgKey = asyncMessageKey("message_"+code, contentType)
								}
							}
						} else {
							data, _ := json.Marshal(mt.Schema)
							var payload interface{}
							_ = json.Unmarshal(data, &payload)
							msg.Payload = payload
							msgKey = asyncMessageKey("message_"+code, contentType)
						}

						// Add to channel
						channelMessages[msgKey] = msg

						// Add Ref to Operation
						escapedMsgKey := strings.ReplaceAll(msgKey, "~", "~0")
						escapedMsgKey = strings.ReplaceAll(escapedMsgKey, "/", "~1")

						opMsgRef := &Message{
							Ref: "#/channels/" + escapedChannelID + "/messages/" + escapedMsgKey,
						}
						op.Messages = append(op.Messages, opMsgRef)
						break
					}
					if len(op.Messages) > 0 {
						break
					}
				}
			}
			doc.Operations[opID] = op
		}

		doc.Channels[channelID] = Channel3{
			Address:  cleanPath,
			Messages: channelMessages,
		}
	}

	if jsonOutput {
		enc := json.NewEncoder(wr)
		enc.SetIndent("", "  ")
		return enc.Encode(doc)
	}

	encoder := yaml.NewEncoder(wr)
	encoder.SetIndent(2)
	return encoder.Encode(doc)
}

func sortedContentTypes(content map[string]model.MediaType) []string {
	contentTypes := make([]string, 0, len(content))
	for contentType := range content {
		contentTypes = append(contentTypes, contentType)
	}
	sort.Strings(contentTypes)
	return contentTypes
}

func asyncMessageKey(prefix, contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return prefix
	}
	return prefix + "_" + sanitizeIdentifier(contentType)
}

func sanitizeIdentifier(value string) string {
	var b strings.Builder
	previousUnderscore := false
	for _, r := range value {
		valid := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if valid {
			b.WriteRune(r)
			previousUnderscore = false
			continue
		}
		if !previousUnderscore {
			b.WriteByte('_')
			previousUnderscore = true
		}
	}
	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "default"
	}
	return result
}
