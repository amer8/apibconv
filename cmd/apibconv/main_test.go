package main

import (
	"flag"
	"os"
	"os/exec"
	"testing"
)

func TestMainFunc(t *testing.T) {
	if os.Getenv("TEST_MAIN_COVERAGE") == "1" {
		// Mock args to avoid actual execution logic but trigger a clean exit path (version check)
		os.Args = []string{"apibconv", "-v"}

		// Reset flag.CommandLine to allow parsing again with our new args
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		// Run main
		main()
		return
	}

	// Run the test as a subprocess to handle os.Exit
	cmd := exec.Command(os.Args[0], "-test.run=TestMainFunc")
	cmd.Env = append(os.Environ(), "TEST_MAIN_COVERAGE=1")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("main() failed with error: %v", err)
	}
}
