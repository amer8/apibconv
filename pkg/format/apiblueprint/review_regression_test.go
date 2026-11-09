package apiblueprint

import (
	"bytes"
	"context"
	"github.com/amer8/apibconv/pkg/model"
	"testing"
)

func TestReviewAllHTTPMethodsRoundTrip(t *testing.T) {
	api := model.NewAPI()
	api.Paths["/things"] = model.PathItem{Head: &model.Operation{}, Options: &model.Operation{}, Trace: &model.Operation{}}
	var out bytes.Buffer
	if err := NewWriter().Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	parsed, err := NewParser().Parse(context.Background(), &out)
	if err != nil {
		t.Fatal(err)
	}
	item := parsed.Paths["/things"]
	if item.Head == nil || item.Options == nil || item.Trace == nil {
		t.Fatalf("lost methods: %#v", item)
	}
}
