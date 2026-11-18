package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSwaggerResponseExamplesAndNoInventedResponses(t *testing.T) {
	source := `swagger: "2.0"
info: {title: Example, version: "1"}
paths:
  /items:
    get:
      produces: [application/json, application/xml]
      responses:
        '200':
          description: OK
          schema: {type: object}
          examples:
            application/json: {id: 7}
            application/xml: '<item/>'
        default: {description: Error}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"2.0", "3.1"} {
		var output bytes.Buffer
		if err := NewWriter(WithJSONOutput(true), WithWriterVersion(version)).Write(context.Background(), api, &output); err != nil {
			t.Fatal(err)
		}
		var doc map[string]interface{}
		if err := json.Unmarshal(output.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		responses := doc["paths"].(map[string]interface{})["/items"].(map[string]interface{})["get"].(map[string]interface{})["responses"].(map[string]interface{})
		if _, ok := responses["404"]; ok {
			t.Fatalf("invented response: %s", output.String())
		}
		if _, ok := doc["definitions"]; ok {
			t.Fatalf("invented definitions: %s", output.String())
		}
		r := responses["200"].(map[string]interface{})
		if version == "2.0" {
			if len(r["examples"].(map[string]interface{})) != 2 {
				t.Fatalf("examples: %s", output.String())
			}
		} else {
			if len(r["content"].(map[string]interface{})) != 2 {
				t.Fatalf("content: %s", output.String())
			}
		}
	}
}
