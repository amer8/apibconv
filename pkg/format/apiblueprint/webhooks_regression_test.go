package apiblueprint

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestWriterRejectsUnrepresentableWebhooks(t *testing.T) {
	api := model.NewAPI()
	api.Webhooks["orderCreated"] = model.PathItem{Post: &model.Operation{}}
	var out bytes.Buffer
	err := NewWriter().Write(context.Background(), api, &out)
	if err == nil || !strings.Contains(err.Error(), "webhooks") || out.Len() != 0 {
		t.Fatalf("expected webhook error and no output, got err=%v output=%q", err, out.String())
	}
}
