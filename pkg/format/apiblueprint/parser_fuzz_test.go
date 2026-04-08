package apiblueprint

import (
	"context"
	"strings"
	"testing"
)

const maxFuzzInputSize = 1 << 20

func FuzzParserParse(f *testing.F) {
	seeds := []string{
		"",
		"FORMAT: 1A\n# Example API\n",
		`FORMAT: 1A
HOST: https://api.example.com

# My API

## Data Structures

## User (object)
+ id (number, required)
+ name (string)

# Group Users

## User [/users/{id}]

### Get User [GET]
+ Response 200 (application/json)
        {
            "id": 1,
            "name": "John Doe"
        }
`,
		"FORMAT: 1A\n# broken [\n+ Response nope\n",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parser := NewParser()
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > maxFuzzInputSize {
			t.Skip()
		}

		_, _ = parser.Parse(ctx, strings.NewReader(input))
	})
}
