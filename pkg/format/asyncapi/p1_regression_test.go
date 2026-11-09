package asyncapi

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/model"
	"go.yaml.in/yaml/v3"
)

func TestP1V2MessageDirectionAndComponents(t *testing.T) {
	input := `asyncapi: 2.6.0
info: {title: Test, version: '1'}
channels:
  events:
    publish: {message: {$ref: '#/components/messages/Event'}}
    subscribe: {message: {$ref: '#/components/messages/Event'}}
components:
  schemas:
    Event: {type: object, properties: {id: {type: string}}}
  messages:
    Event: {payload: {$ref: '#/components/schemas/Event'}}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if s := api.Components.Schemas["Event"]; s == nil || s.Ref != "" || s.Properties["id"] == nil {
		t.Fatalf("component schema = %#v", s)
	}
	item := api.Paths["events"]
	if item.Get == nil || item.Post == nil {
		t.Fatalf("operations = %#v", item)
	}
	if got := item.Get.Responses["200"].Content["application/json"].Schema.Ref; got != "#/components/schemas/Event" {
		t.Fatalf("receive ref = %q", got)
	}
	if got := item.Post.RequestBody.Content["application/json"].Schema.Ref; got != "#/components/schemas/Event" {
		t.Fatalf("send ref = %q", got)
	}
	for _, version := range []string{"2.6", "3.0"} {
		var out bytes.Buffer
		if err := NewWriter(WithAsyncWriterVersion(version)).Write(context.Background(), api, &out); err != nil {
			t.Fatal(err)
		}
		var doc map[string]interface{}
		if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if version == "3.0" {
			ops := doc["operations"].(map[string]interface{})
			if ops["receive_events"].(map[string]interface{})["action"] != "receive" || ops["send_events"].(map[string]interface{})["action"] != "send" {
				t.Fatalf("operations: %v", ops)
			}
		}
		round, err := NewParser().Parse(context.Background(), &out)
		if err != nil {
			t.Fatal(err)
		}
		schema := round.Components.Schemas["Event"]
		if schema == nil || schema.Ref != "" || schema.Properties["id"] == nil {
			t.Fatalf("%s roundtrip schema %#v", version, schema)
		}
		op := round.Paths["events"]
		if op.Get == nil || op.Post == nil || op.Get.Responses["200"].Content["application/json"].Schema == nil || op.Post.RequestBody == nil {
			t.Fatalf("%s roundtrip operations %#v", version, op)
		}
	}
}

func TestP1RejectInvalidOutputVersionAndEncoding(t *testing.T) {
	for _, tc := range []struct{ version, encoding string }{{"5.0", "json"}, {"3.0", "potato"}} {
		var out bytes.Buffer
		err := NewWriter(WithAsyncWriterVersion(tc.version)).Write(converter.WithEncoding(context.Background(), tc.encoding), model.NewAPI(), &out)
		if err == nil || out.Len() != 0 {
			t.Fatalf("version %q encoding %q: err=%v output=%s", tc.version, tc.encoding, err, out.String())
		}
	}
}

func TestP1V3InlineAndReferencedMessages(t *testing.T) {
	input := `asyncapi: 3.0.0
info: {title: Test, version: '1'}
channels:
  events:
    address: /events
    messages:
      Event: {$ref: '#/components/messages/Event'}
operations:
  receiveEvent:
    action: receive
    channel: {$ref: '#/channels/events'}
    messages: [{$ref: '#/channels/events/messages/Event'}]
components:
  schemas:
    Event: {type: object, properties: {id: {type: string}}}
  messages:
    Event: {contentType: application/json, payload: {$ref: '#/components/schemas/Event'}}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	op := api.Paths["/events"].Get
	if op == nil || op.Responses["200"].Content["application/json"].Schema.Ref != "#/components/schemas/Event" || api.Components.Schemas["Event"].Properties["id"] == nil {
		t.Fatalf("parsed operation: %#v", op)
	}
	var out bytes.Buffer
	if err = NewWriter(WithAsyncWriterVersion("3.0")).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	round, err := NewParser().Parse(context.Background(), &out)
	if err != nil {
		t.Fatal(err)
	}
	if round.Paths["/events"].Get.Responses["200"].Content["application/json"].Schema == nil {
		t.Fatal("roundtrip lost message schema")
	}
}

func TestP1RejectLossyOperations(t *testing.T) {
	input := `asyncapi: 3.0.0
info: {title: Test, version: '1'}
channels: {events: {address: /events}}
operations:
  a: {action: send, channel: {$ref: '#/channels/events'}}
  b: {action: send, channel: {$ref: '#/channels/events'}}
`
	if _, err := NewParser().Parse(context.Background(), strings.NewReader(input)); err == nil || !strings.Contains(err.Error(), "multiple send") {
		t.Fatalf("duplicate operations error = %v", err)
	}
	for _, version := range []string{"2.6", "3.0"} {
		api := model.NewAPI()
		api.Paths["/x"] = model.PathItem{Put: &model.Operation{}}
		var out bytes.Buffer
		err := NewWriter(WithAsyncWriterVersion(version)).Write(context.Background(), api, &out)
		if err == nil || out.Len() != 0 {
			t.Fatalf("%s error=%v output=%q", version, err, out.String())
		}
	}
}
