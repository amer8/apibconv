package main

import (
	"strings"
	"testing"
)

func TestPlatformSupport(t *testing.T) {
	for _, goos := range []string{"linux", "darwin"} {
		if err := checkPlatform(goos); err != nil {
			t.Errorf("%s should be supported: %v", goos, err)
		}
	}
	if err := checkPlatform("windows"); err == nil || !strings.Contains(err.Error(), "WSL") {
		t.Fatalf("Windows must be rejected with WSL guidance, got %v", err)
	}
}
