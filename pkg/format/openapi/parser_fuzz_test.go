package openapi

import (
	"bytes"
	"context"
	"testing"
)

const maxFuzzInputSize = 1 << 20

func FuzzParserParse(f *testing.F) {
	seeds := [][]byte{
		{},
		[]byte(`openapi: 3.0.0
info:
  title: Sample API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get all users
      responses:
        '200':
          description: A list of users
`),
		[]byte(`swagger: "2.0"
info:
  title: Swagger Petstore
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
`),
		[]byte("openapi: ["),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parser := NewParser()
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > maxFuzzInputSize {
			t.Skip()
		}

		_, _ = parser.Parse(ctx, bytes.NewReader(input))
	})
}
