package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestWebhookOutputVersionSupport(t *testing.T) {
	api := model.NewAPI()
	api.Info.Title = "Webhook API"
	api.Info.Version = "1"
	api.Webhooks["orderCreated"] = model.PathItem{Post: &model.Operation{Responses: model.Responses{"200": {Description: "OK"}}}}
	for _, tc := range []struct {
		version string
		wantErr bool
	}{
		{"2.0", true}, {"2.0.0", true}, {"3.0", true}, {"3.0.3", true}, {"3.1", false}, {"3.1.0", false},
	} {
		t.Run(tc.version, func(t *testing.T) {
			var out bytes.Buffer
			err := NewWriter(WithWriterVersion(tc.version)).Write(context.Background(), api, &out)
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "webhooks") || out.Len() != 0 {
					t.Fatalf("expected webhook error without output, got err=%v output=%q", err, out.String())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Webhooks map[string]json.RawMessage `json:"webhooks"`
			}
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			if _, ok := doc.Webhooks["orderCreated"]; !ok {
				t.Fatalf("missing webhook: %s", out.String())
			}
		})
	}
}
