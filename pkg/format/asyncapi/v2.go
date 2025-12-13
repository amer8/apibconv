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

// parseV2 implements AsyncAPI 2.x parsing
func (p *Parser) parseV2(data []byte) (*model.API, error) {
	var doc AsyncAPI2
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("asyncapi v2 parse failed: %w", err)
	}

	api := model.NewAPI()
	api.Version = doc.AsyncAPI
	api.Info.Title = doc.Info.Title
	api.Info.Version = doc.Info.Version
	api.Info.Description = doc.Info.Description

	// Sort servers by name for deterministic output
	serverNames := make([]string, 0, len(doc.Servers))
	for k := range doc.Servers {
		serverNames = append(serverNames, k)
	}
	sort.Strings(serverNames)

	for _, name := range serverNames {
		s := doc.Servers[name]
		api.Servers = append(api.Servers, model.Server{
			URL:         s.URL,
			Description: s.Protocol,
			Bindings:    s.Bindings,
		})
	}
	
	// Convert Components
	// TODO: map other components if needed

	for path, ch := range doc.Channels {
		// Normalize path
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		pi := model.PathItem{}
		pi.Parameters = p.convertParameters(ch.Parameters)

		if ch.Publish != nil {
			op := &model.Operation{
				OperationID: ch.Publish.OperationID,
				Summary:     ch.Publish.Summary,
				Description: ch.Publish.Description,
				Bindings:    ch.Publish.Bindings,
			}
			
			// Map Message to RequestBody
			if ch.Publish.Message != nil {
				op.RequestBody = &model.RequestBody{
					Content: map[string]model.MediaType{
						"application/json": { // Defaulting
							Schema: p.convertSchema(ch.Publish.Message),
						},
					},
				}
			}
			
			// Add default response for Publish
			op.Responses = model.Responses{
				"200": model.Response{Description: "OK"},
			}
			
			pi.Post = op
		}

		if ch.Subscribe != nil {
			op := &model.Operation{
				OperationID: ch.Subscribe.OperationID,
				Summary:     ch.Subscribe.Summary,
				Description: ch.Subscribe.Description,
				Bindings:    ch.Subscribe.Bindings,
			}

			// Map Message to Responses
			schema := p.convertSchema(ch.Subscribe.Message)
			op.Responses = model.Responses{
				"200": model.Response{
					Description: "OK",
					Content: map[string]model.MediaType{
						"application/json": {
							Schema: schema,
						},
					},
				},
			}
			pi.Get = op
		}

		api.AddPath(path, &pi)
	}

	return api, nil
}

func (p *Parser) convertParameters(params map[string]Parameter) []model.Parameter {
	result := make([]model.Parameter, 0, len(params))
	for name, param := range params {
		mp := model.Parameter{
			Name:        name,
			In:          "path", // Channels usually define path parameters
			Description: param.Description,
			Required:    true, // Path params are always required
		}
		
		if param.Schema != nil {
			s := p.convertSchemaPayload(param.Schema)
			mp.Schema = s
		}
		result = append(result, mp)
	}
	return result
}

func (p *Parser) convertSchema(msg *Message) *model.Schema {
	if msg == nil {
		return nil
	}
	
	// If message itself is a ref
	if msg.Ref != "" {
		return &model.Schema{Ref: msg.Ref}
	}
	
	// If payload is present
	if msg.Payload != nil {
		return p.convertSchemaPayload(msg.Payload)
	}
	
	return nil
}

func (p *Parser) convertSchemaPayload(payload interface{}) *model.Schema {
	// Use JSON roundtrip to convert map[string]interface{} to model.Schema
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	
	var schema model.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil
	}
	return &schema
}


// writeV2 implements AsyncAPI 2.x writing
func (w *Writer) writeV2(api *model.API, wr io.Writer, targetProtocol string, jsonOutput bool) error {
	doc := AsyncAPI2{
		AsyncAPI:   w.version,
		Channels:   make(map[string]Channel2),
		Servers:    make(map[string]Server),
		Components: make(map[string]interface{}),
	}

	doc.Info.Title = api.Info.Title
	doc.Info.Version = api.Info.Version
	doc.Info.Description = api.Info.Description

	// Servers
	for i, s := range api.Servers {
		name := fmt.Sprintf("server%d", i)
		var protocol string

		switch {
		case targetProtocol != "" && targetProtocol != "auto":
			protocol = targetProtocol
		case targetProtocol == "auto":
			protocol = detect.Protocol(s.URL)
		case s.Description != "" && len(s.Description) < 10: // Check if description is a short string, likely a protocol
			protocol = s.Description
		default:
			protocol = detect.Protocol(s.URL)
		}

		doc.Servers[name] = Server{
			URL:      s.URL,
			Protocol: protocol,
			Bindings: s.Bindings,
		}
	}

	// Components -> Messages (simplified mapping)
	messages := make(map[string]interface{})
	schemas := make(map[string]interface{})

	if len(api.Components.Schemas) > 0 {
		for name, schema := range api.Components.Schemas {
			// Convert model.Schema to interface{}
			data, _ := json.Marshal(schema)
			var schemaMap map[string]interface{}
			_ = json.Unmarshal(data, &schemaMap)
			schemas[name] = schemaMap

			// Also create a message wrapper for it
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

	// Channels
	paths := make([]string, 0, len(api.Paths))
	for path := range api.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := api.Paths[path]
		ch := Channel2{}

		if item.Post != nil {
			op := &Operation{
				OperationID: item.Post.OperationID,
				Summary:     item.Post.Summary,
				Description: item.Post.Description,
				Bindings:    item.Post.Bindings,
			}

			// Try to find payload
			if item.Post.RequestBody != nil {
				for _, mt := range item.Post.RequestBody.Content {
					if mt.Schema != nil {
						msg := &Message{}
						if mt.Schema.Ref != "" {
							// Rewrite ref to message if possible, or schema
							ref := mt.Schema.Ref
							if strings.HasPrefix(ref, "#/components/schemas/") {
								// In AsyncAPI, we often refer to messages.
								// If we created a message wrapper above, point to it.
								name := strings.TrimPrefix(ref, "#/components/schemas/")
								msg.Ref = "#/components/messages/" + name
							} else {
								msg.Ref = ref
							}
						} else {
							// Inline payload
							data, _ := json.Marshal(mt.Schema)
							var payload interface{}
							_ = json.Unmarshal(data, &payload)
							msg.Payload = payload
						}
						op.Message = msg
						break // Only one message
					}
				}
			}
			ch.Publish = op
		}

		if item.Get != nil {
			op := &Operation{
				OperationID: item.Get.OperationID,
				Summary:     item.Get.Summary,
				Description: item.Get.Description,
				Bindings:    item.Get.Bindings,
			}

			// Try to find response payload
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
							op.Message = msg
							break
						}
					}
					if op.Message != nil {
						break
					}
				}
			}
			ch.Subscribe = op
		}

		if ch.Publish != nil || ch.Subscribe != nil {
			doc.Channels[path] = ch
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
