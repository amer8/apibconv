package validator

import (
	"context"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

type mixedRule struct{}

func (mixedRule) Name() string                  { return "mixed" }
func (mixedRule) Level() format.ValidationLevel { return format.LevelError }
func (mixedRule) Validate(context.Context, *model.API) []format.ValidationError {
	return []format.ValidationError{{Level: format.LevelWarning}, {Level: format.LevelError}, {Level: format.LevelInfo}}
}

func TestWithWarnings(t *testing.T) {
	for _, tc := range []struct {
		include bool
		want    int
	}{{true, 3}, {false, 2}} {
		v := New(WithWarnings(tc.include))
		v.AddRule(mixedRule{})
		issues, err := v.Validate(context.Background(), model.NewAPI())
		if err != nil || len(issues) != tc.want {
			t.Fatalf("WithWarnings(%v): issues=%v err=%v", tc.include, issues, err)
		}
		if !tc.include && (issues[0].Level != format.LevelError || issues[1].Level != format.LevelInfo) {
			t.Fatalf("non-warning issues lost: %v", issues)
		}
	}
}
