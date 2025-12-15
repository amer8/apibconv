package asyncapi

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

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

	// Servers (Sorted)
	serverNames := make([]string, 0, len(doc.Servers))
	for k := range doc.Servers {
		serverNames = append(serverNames, k)
	}
	sort.Strings(serverNames)

	for _, name := range serverNames {
		s := doc.Servers[name]
		api.Servers = append(api.Servers, model.Server{
			Name:        name,
			URL:         s.Host,
			Protocol:    s.Protocol,
			Description: s.Description,
			Bindings:    s.Bindings,
		})
	}

	// Map Channels by ID for lookup (needed for operations)
	channelMap := make(map[string]string) // ID -> Address
	for id, ch := range doc.Channels {
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
					chRef = strings.TrimPrefix(ref, "#/channels/")
				} else {
					parts := strings.Split(ref, "/")
					if len(parts) > 0 {
						chRef = parts[len(parts)-1]
					}
				}
			}
		}

		if chRef == "" {
			continue
		}

		if addr, ok := channelMap[chRef]; ok {
			path = addr
		} else {
			path = chRef // Unknown ID, use as path
		}

		// Get or Create PathItem
		pi, _ := api.GetPath(path)

		modelOp := &model.Operation{
			OperationID: op.OperationID,
			Summary:     op.Summary,
			Description: op.Description,
			Bindings:    op.Bindings,
		}
		if modelOp.OperationID == "" {
			modelOp.OperationID = opID
		}

		switch op.Action {
		case "send":
			pi.Post = modelOp
		case "receive":
			pi.Get = modelOp
		}

		api.AddPath(path, &pi)
	}

	return api, nil
}

// writeV3 implements AsyncAPI 3.0 writing
func (w *Writer) writeV3(api *model.API, wr io.Writer, targetProtocol string, jsonOutput bool) error {
	doc := AsyncAPI3{
		AsyncAPI:   w.version,
		Channels:   make(map[string]Channel3),
		Operations: make(map[string]Operation3),
		Servers:    make(map[string]ServerV3),
		Components: make(map[string]interface{}),
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
		// Generate Channel ID
		channelID := item.Name
		if channelID == "" {
			channelID = path
		}

		doc.Channels[channelID] = Channel3{
			Address: path,
		}

		if item.Post != nil {
			opID := item.Post.OperationID
			if opID == "" {
				opID = "send_" + channelID
			}
			op := Operation3{
				Action:      "send",
				Channel:     map[string]string{"$ref": "#/channels/" + channelID},
				Summary:     item.Post.Summary,
				Description: item.Post.Description,
				OperationID: item.Post.OperationID,
				Bindings:    item.Post.Bindings,
			}

			// Add Messages
			if item.Post.RequestBody != nil {
				for _, mt := range item.Post.RequestBody.Content {
					if mt.Schema != nil {
						msg := &Message{}
						if mt.Schema.Ref != "" {
							ref := mt.Schema.Ref
							if strings.HasPrefix(ref, "#/components/schemas/") {
								name := strings.TrimPrefix(ref, "#/components/schemas/")
								msg.Ref = "#/components/messages/" + name
							} else {
								msg.Ref = ref
							}
						} else {
							data, _ := json.Marshal(mt.Schema)
							var payload interface{}
							_ = json.Unmarshal(data, &payload)
							msg.Payload = payload
						}
						op.Messages = append(op.Messages, msg)
						break
					}
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
				Channel:     map[string]string{"$ref": "#/channels/" + channelID},
				Summary:     item.Get.Summary,
				Description: item.Get.Description,
				OperationID: item.Get.OperationID,
				Bindings:    item.Get.Bindings,
			}

			// Add Messages
			if len(item.Get.Responses) > 0 {
				var codes []string
				for code := range item.Get.Responses {
					codes = append(codes, code)
				}
				sort.Strings(codes)

				for _, code := range codes {
					resp := item.Get.Responses[code]
					for _, mt := range resp.Content {
						if mt.Schema != nil {
							msg := &Message{}
							if mt.Schema.Ref != "" {
								ref := mt.Schema.Ref
								if strings.HasPrefix(ref, "#/components/schemas/") {
									name := strings.TrimPrefix(ref, "#/components/schemas/")
									msg.Ref = "#/components/messages/" + name
								} else {
									msg.Ref = ref
								}
							} else {
								data, _ := json.Marshal(mt.Schema)
								var payload interface{}
								_ = json.Unmarshal(data, &payload)
								msg.Payload = payload
							}
							op.Messages = append(op.Messages, msg)
							break
						}
					}
					if len(op.Messages) > 0 {
						break
					}
				}
			}
			doc.Operations[opID] = op
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
