package asyncapi

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestDefaultContentTypeOnReferencedMessages(t *testing.T) {
	for _, tc := range []struct {
		name, documentDefault, messageType, want string
	}{
		{"document default", "application/xml", "", "application/xml"},
		{"message overrides default", "application/xml", "application/cbor", "application/cbor"},
		{"legacy fallback", "", "", "application/json"},
	} {
		for _, version := range []string{"2.6.0", "3.0.0"} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				var defaultLine, contentLine string
				if tc.documentDefault != "" {
					defaultLine = "defaultContentType: " + tc.documentDefault + "\n"
				}
				if tc.messageType != "" {
					contentLine = "      contentType: " + tc.messageType + "\n"
				}
				input := fmt.Sprintf("asyncapi: %s\ninfo: {title: Test, version: '1'}\n%scomponents:\n  messages:\n    Event:\n%s      payload: {type: string}\n", version, defaultLine, contentLine)
				if version == "2.6.0" {
					input += "channels:\n  events:\n    publish: {message: {$ref: '#/components/messages/Event'}}\n    subscribe: {message: {$ref: '#/components/messages/Event'}}\n"
				} else {
					input += "channels:\n  events:\n    address: events\n    messages:\n      event: {$ref: '#/components/messages/Event'}\noperations:\n  sendEvent:\n    action: send\n    channel: {$ref: '#/channels/events'}\n    messages:\n      - {$ref: '#/channels/events/messages/event'}\n  receiveEvent:\n    action: receive\n    channel: {$ref: '#/channels/events'}\n    messages:\n      - {$ref: '#/channels/events/messages/event'}\n"
				}
				api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
				if err != nil {
					t.Fatal(err)
				}
				checkMessages := func(api *model.API) {
					t.Helper()
					item := api.Paths["events"]
					if item.Post == nil || item.Post.RequestBody == nil || item.Get == nil {
						t.Fatalf("missing operations or message: %#v", item)
					}
					for name, content := range map[string]map[string]model.MediaType{
						"send":    item.Post.RequestBody.Content,
						"receive": item.Get.Responses["200"].Content,
					} {
						media, ok := content[tc.want]
						if !ok || len(content) != 1 || media.Schema == nil || media.Schema.Type != model.TypeString {
							t.Fatalf("%s: expected %s string payload, got %#v", name, tc.want, content)
						}
					}
				}
				checkMessages(api)
				var out bytes.Buffer
				if err := NewWriter(WithAsyncWriterVersion(version)).Write(context.Background(), api, &out); err != nil {
					t.Fatal(err)
				}
				roundTrip, err := NewParser().Parse(context.Background(), &out)
				if err != nil {
					t.Fatal(err)
				}
				checkMessages(roundTrip)
			})
		}
	}
}

func TestV3ChannelCollisionIncludesOperationlessPaths(t *testing.T) {
	for _, tc := range []struct{ activePath, emptyPath string }{
		{"/a", "/z"}, {"/z", "/a"},
	} {
		t.Run(tc.activePath, func(t *testing.T) {
			api := model.NewAPI()
			api.Paths[tc.activePath] = model.PathItem{Name: "same", Post: &model.Operation{}}
			api.Paths[tc.emptyPath] = model.PathItem{Name: "same"}
			var out bytes.Buffer
			err := NewWriter(WithAsyncWriterVersion("3.0")).Write(context.Background(), api, &out)
			if err == nil || !strings.Contains(err.Error(), "same channel") || out.Len() != 0 {
				t.Fatalf("expected collision error without output, got %v; output %q", err, out.String())
			}
		})
	}
}

func TestMessageTraitsSetContentType(t *testing.T) {
	for _, tc := range []struct {
		name, version, traits, explicit, want string
	}{
		{"v2 inline", "2.6.0", "[{contentType: application/cbor}]", "", "application/cbor"},
		{"v2 ref and last trait wins", "2.6.0", "[{$ref: '#/components/messageTraits/Encoded'}, {contentType: text/plain}]", "", "text/plain"},
		{"v2 trait overrides message", "2.6.0", "[{$ref: '#/components/messageTraits/Encoded'}]", "application/json", "application/cbor"},
		{"v3 inline", "3.0.0", "[{contentType: application/cbor}]", "", "application/cbor"},
		{"v3 reference", "3.0.0", "[{$ref: '#/components/messageTraits/Encoded'}]", "", "application/cbor"},
		{"v3 first trait wins", "3.0.0", "[{$ref: '#/components/messageTraits/Encoded'}, {contentType: text/plain}]", "", "application/cbor"},
		{"v3 message overrides trait", "3.0.0", "[{$ref: '#/components/messageTraits/Encoded'}]", "application/json", "application/json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			messageType := ""
			if tc.explicit != "" {
				messageType = "      contentType: " + tc.explicit + "\n"
			}
			input := fmt.Sprintf("asyncapi: %s\ninfo: {title: T, version: '1'}\ndefaultContentType: application/xml\ncomponents:\n  messageTraits:\n    Encoded: {contentType: application/cbor}\n  messages:\n    Event:\n%s      traits: %s\n      payload: {type: string}\n", tc.version, messageType, tc.traits)
			if tc.version == "2.6.0" {
				input += "channels:\n  events:\n    subscribe: {message: {$ref: '#/components/messages/Event'}}\n"
			} else {
				input += "channels:\n  events:\n    address: events\n    messages:\n      event: {$ref: '#/components/messages/Event'}\noperations:\n  sendEvent:\n    action: send\n    channel: {$ref: '#/channels/events'}\n    messages:\n      - {$ref: '#/channels/events/messages/event'}\n"
			}
			api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			content := api.Paths["events"].Post.RequestBody.Content
			if _, ok := content[tc.want]; !ok || len(content) != 1 {
				t.Fatalf("expected %s; got %#v", tc.want, content)
			}
		})
	}
}

func TestMessageTraitReferenceMustResolve(t *testing.T) {
	input := `asyncapi: 2.6.0
info: {title: T, version: '1'}
channels:
  events:
    subscribe:
      message:
        traits: [{$ref: '#/components/messageTraits/Missing'}]
        payload: {type: string}
`
	_, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "trait") {
		t.Fatalf("expected unresolved trait error, got %v", err)
	}
}
