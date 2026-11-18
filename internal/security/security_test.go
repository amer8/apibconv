package security

import (
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestCustomFormatChannelServerSelectionIsNotSilentlyLost(t *testing.T) {
	api := model.NewAPI()
	api.Paths["/events"] = model.PathItem{ChannelServers: []string{"production"}}
	if err := CheckChannelServers(api, "OpenAPI"); err == nil || !strings.Contains(err.Error(), "/events") {
		t.Fatalf("expected channel restriction to be rejected, got %v", err)
	}
}
