package asyncapi

import (
	"bytes"
	"context"
	"testing"
)

const maxFuzzInputSize = 1 << 20

func FuzzParserParse(f *testing.F) {
	seeds := [][]byte{
		{},
		[]byte(`asyncapi: 2.6.0
info:
  title: User API
  version: 1.0.0
channels:
  user/signedup:
    publish:
      operationId: userSignedUp
      message:
        payload:
          type: object
          properties:
            userId:
              type: string
`),
		[]byte(`asyncapi: 3.0.0
info:
  title: Streetlights API
  version: 1.0.0
channels:
  lightingMeasured:
    address: smartylighting/streetlights/1/0/event/lighting/measured
operations:
  receiveLightMeasurement:
    action: receive
    channel:
      $ref: '#/channels/lightingMeasured'
`),
		[]byte("invalid: - yaml"),
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
