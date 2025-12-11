package cli

import (
	"flag"
	"os"
	"testing"
)

func TestRun(t *testing.T) {
	// Save original state
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine

	// Restore original state after tests
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	tests := []struct {
		name     string
		args     []string
		wantExit int
		setup    func(t *testing.T) (cleanup func())
	}{
		{
			name:     "Version flag",
			args:     []string{"-v"},
			wantExit: 0,
		},
		{
			name:     "Help flag",
			args:     []string{"-h"},
			wantExit: 0,
		},
		{
			name:     "Missing arguments",
			args:     []string{},
			wantExit: 1, // Expect failure because no input/output provided
		},
		{
			name:     "Validate file",
			args:     []string{"--validate", "-f", "testdata/valid_openapi.json"},
			wantExit: 0,
			setup: func(t *testing.T) func() {
				// Create a dummy valid file
				content := `{"openapi": "3.0.0", "info": {"title": "Test", "version": "1.0"}, "paths": {}}`
				_ = os.MkdirAll("testdata", 0o755)
				_ = os.WriteFile("testdata/valid_openapi.json", []byte(content), 0o644)
				return func() {
					_ = os.RemoveAll("testdata")
				}
			},
		},
		{
			name:     "Convert APIB to OpenAPI",
			args:     []string{"-f", "testdata/input.apib", "-o", "testdata/output.json"},
			wantExit: 0,
			setup: func(t *testing.T) func() {
				content := "FORMAT: 1A\n# Test API\n## GET /"
				_ = os.MkdirAll("testdata", 0o755)
				_ = os.WriteFile("testdata/input.apib", []byte(content), 0o644)
				return func() {
					_ = os.RemoveAll("testdata")
				}
			},
		},
		{
			name:     "Convert OpenAPI to APIB",
			args:     []string{"-f", "testdata/input.json", "-o", "testdata/output.apib"},
			wantExit: 0,
			setup: func(t *testing.T) func() {
				content := `{"openapi": "3.0.0", "info": {"title": "Test", "version": "1.0"}, "paths": {}}`
				_ = os.MkdirAll("testdata", 0o755)
				_ = os.WriteFile("testdata/input.json", []byte(content), 0o644)
				return func() {
					_ = os.RemoveAll("testdata")
				}
			},
		},
		{
			name:     "Convert APIB to AsyncAPI (Missing Protocol)",
			args:     []string{"-f", "testdata/input.apib", "-o", "testdata/output.json", "--to", "asyncapi"},
			wantExit: 1, // Protocol required
			setup: func(t *testing.T) func() {
				content := "FORMAT: 1A\n# Test API\n## GET /"
				_ = os.MkdirAll("testdata", 0o755)
				_ = os.WriteFile("testdata/input.apib", []byte(content), 0o644)
				return func() {
					_ = os.RemoveAll("testdata")
				}
			},
		},
		{
			name:     "Invalid Output Format",
			args:     []string{"-f", "testdata/input.apib", "-o", "testdata/output.txt", "--to", "invalid"},
			wantExit: 1,
			setup: func(t *testing.T) func() {
				content := "FORMAT: 1A\n# Test API\n## GET /"
				_ = os.MkdirAll("testdata", 0o755)
				_ = os.WriteFile("testdata/input.apib", []byte(content), 0o644)
				return func() {
					_ = os.RemoveAll("testdata")
				}
			},
		},
		{
			name:     "Convert AsyncAPI to APIB",
			args:     []string{"-f", "testdata/input.json", "-o", "testdata/output.apib"},
			wantExit: 0,
			setup: func(t *testing.T) func() {
				content := `{"asyncapi": "2.6.0", "info": {"title": "Test", "version": "1.0"}, "channels": {}}`
				_ = os.MkdirAll("testdata", 0o755)
				_ = os.WriteFile("testdata/input.json", []byte(content), 0o644)
				return func() {
					_ = os.RemoveAll("testdata")
				}
			},
		},
		{
			name:     "Output as YAML",
			args:     []string{"-f", "testdata/input.apib", "-o", "testdata/output.yaml", "-e", "yaml"},
			wantExit: 0,
			setup: func(t *testing.T) func() {
				content := "FORMAT: 1A\n# Test API\n## GET /"
				_ = os.MkdirAll("testdata", 0o755)
				_ = os.WriteFile("testdata/input.apib", []byte(content), 0o644)
				return func() {
					_ = os.RemoveAll("testdata")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				cleanup := tt.setup(t)
				defer cleanup()
			}

			// Reset global variables
			inputFile = ""
			outputFile = ""
			openapiVersion = "3.0"
			asyncapiVersion = "2.6"
			outputFormat = ""
			encodingFormat = ""
			protocol = ""
			validateOnly = false
			showVersion = false
			showHelp = false

			// Reset flag command line
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			// Mock Usage to avoid pollution
			flag.Usage = func() {}

			// Set args
			os.Args = append([]string{"apibconv"}, tt.args...)

			// Capture stdout/stderr to avoid noise
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			// We can direct them to /dev/null or a pipe if we want to check output
			// For now, just discard
			null, err := os.Open(os.DevNull)
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = null
			os.Stderr = null
			defer func() {
				_ = null.Close()
				os.Stdout = oldStdout
				os.Stderr = oldStderr
			}()

			exitCode := Run("0.0.0-test")
			if exitCode != tt.wantExit {
				t.Errorf("Run() exit code = %d, want %d", exitCode, tt.wantExit)
			}
		})
	}
}

func TestRun_Stdin(t *testing.T) {
	// Test reading from stdin
	// We need to simulate stdin.
	// However, the Run function checks for os.ModeCharDevice on os.Stdin.Stat().
	// This makes it hard to test stdin redirection in a unit test without mocking os.Stdin which is not easily done in Go.
	// But we can skip the ModeCharDevice check if we were using a custom interface, but we are using os package directly.

	// If we pipe in the test runner, the ModeCharDevice bit is cleared.
	// Let's try to mock stdin by creating a pipe.

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	os.Stdin = r

	// Write content to pipe
	go func() {
		_, _ = w.WriteString(`{"openapi": "3.0.0", "info": {"title": "Test", "version": "1.0"}, "paths": {}}`)
		_ = w.Close()
	}()

	// Setup test environment
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	inputFile = ""
	outputFile = ""
	validateOnly = false

	// Set args to validate only (reads from stdin if no file)
	os.Args = []string{"apibconv", "--validate"}

	// Capture output
	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull) // discard stdout
	defer func() { os.Stdout = oldStdout }()

	// Run
	exitCode := Run("test")

	if exitCode != 0 {
		t.Errorf("Run() with stdin exit code = %d, want 0", exitCode)
	}
}
