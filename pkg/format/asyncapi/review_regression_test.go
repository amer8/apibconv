package asyncapi

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/specschema"
	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
	"go.yaml.in/yaml/v3"
)

func TestReviewUnsecuredCollisionsFail(t *testing.T) {
	for _, tc := range []struct {
		name, version string
		paths         map[string]model.PathItem
	}{
		{"v3 channel", "3.0", map[string]model.PathItem{"/a": {Name: "same", Post: &model.Operation{}}, "/b": {Name: "same", Post: &model.Operation{}}}},
		{"v3 operation", "3.0", map[string]model.PathItem{"/a": {Post: &model.Operation{OperationID: "same"}}, "/b": {Post: &model.Operation{OperationID: "same"}}}},
		{"v2 channel", "2.6", map[string]model.PathItem{"/a{?query}": {Post: &model.Operation{}}, "/a": {Get: &model.Operation{}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			api.Paths = tc.paths
			var out bytes.Buffer
			if err := NewWriter(WithAsyncWriterVersion(tc.version)).Write(context.Background(), api, &out); err == nil || out.Len() != 0 {
				t.Fatalf("collision: err=%v output=%q", err, out.String())
			}
		})
	}
}

func TestReviewV3TwoSelectedMessagesRoundTrip(t *testing.T) {
	api := model.NewAPI()
	api.Info.Title = "Test"
	api.Info.Version = "1"
	api.Paths["/events"] = model.PathItem{
		Post: &model.Operation{RequestBody: &model.RequestBody{Content: map[string]model.MediaType{"application/json": {Schema: &model.Schema{Type: model.TypeString}}}}},
		Get:  &model.Operation{Responses: model.Responses{"200": {Content: map[string]model.MediaType{"application/json": {Schema: &model.Schema{Type: model.TypeInteger}}}}}},
	}
	var out bytes.Buffer
	if err := NewWriter(WithAsyncWriterVersion("3.0")).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	parsed, err := NewParser().Parse(context.Background(), &out)
	if err != nil {
		t.Fatal(err)
	}
	item := parsed.Paths["/events"]
	if item.Post == nil || item.Get == nil || item.Post.RequestBody.Content["application/json"].Schema.Type != model.TypeString || item.Get.Responses["200"].Content["application/json"].Schema.Type != model.TypeInteger {
		t.Fatalf("lost messages: %#v", item)
	}
}

func TestReviewV3ServerRoundTrip(t *testing.T) {
	for _, tc := range []struct{ url, protocol string }{
		{"https://example.com/api/v1", "https"}, {"wss://example.com/socket", "wss"}, {"mqtts://broker.example/v1", "mqtts"}, {"amqps://broker.example/vhost", "amqps"},
	} {
		t.Run(tc.protocol, func(t *testing.T) {
			api := model.NewAPI()
			api.Info.Title = "Test"
			api.Info.Version = "1"
			api.Servers = []model.Server{{URL: tc.url}}
			var out bytes.Buffer
			ctx := converter.WithProtocol(context.Background(), "auto")
			if err := NewWriter(WithAsyncWriterVersion("3.0")).Write(ctx, api, &out); err != nil {
				t.Fatal(err)
			}
			if issues := specschema.Validate(format.FormatAsyncAPI, out.Bytes()); len(issues) != 0 {
				t.Fatalf("invalid output: %v\n%s", issues, out.String())
			}
			var doc AsyncAPI3
			if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			server := doc.Servers["server0"]
			if server.Protocol != tc.protocol || strings.Contains(server.Host, "://") || server.Pathname == "" {
				t.Fatalf("server = %#v", server)
			}
			parsed, err := NewParser().Parse(context.Background(), &out)
			if err != nil {
				t.Fatal(err)
			}
			if got := parsed.Servers[0].URL; got != tc.url {
				t.Fatalf("URL=%q want %q", got, tc.url)
			}
		})
	}
}
