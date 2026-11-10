package converter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// --- Mocks ---

type mockParser struct {
	fmt format.Format
	api *model.API
	err error
}

func TestValidationOptionsRejectInvalidModels(t *testing.T) {
	for _, tc := range []struct {
		name           string
		validateInput  bool
		validateOutput bool
		wantErr        bool
	}{
		{name: "validation disabled", wantErr: false},
		{name: "input validation", validateInput: true, wantErr: true},
		{name: "output validation", validateOutput: true, wantErr: true},
		{name: "both validations", validateInput: true, validateOutput: true, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			api.Paths["bad-path"] = model.PathItem{}
			api.Components.Schemas["BadType"] = &model.Schema{Type: "not-a-type"}
			c, err := New(WithValidation(tc.validateInput, tc.validateOutput))
			if err != nil {
				t.Fatal(err)
			}
			c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: api})
			c.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
			var output bytes.Buffer
			err = c.Convert(context.Background(), bytes.NewReader(nil), &output, format.FormatOpenAPI, format.FormatOpenAPI)
			if !tc.wantErr {
				if err != nil || output.String() != "mock output" {
					t.Fatalf("Convert() = %q, %v; want mock output and no error", output.String(), err)
				}
				return
			}
			if !errors.Is(err, apierrors.ErrValidationFailed) {
				t.Fatalf("Convert() error = %v, want ErrValidationFailed", err)
			}
			if !strings.Contains(err.Error(), "bad-path") || !strings.Contains(err.Error(), "BadType") {
				t.Fatalf("Convert() error omits validation details: %v", err)
			}
			var issue format.ValidationError
			if !errors.As(err, &issue) {
				t.Fatalf("Convert() error does not expose validation issue: %v", err)
			}
			if output.Len() != 0 {
				t.Fatalf("invalid model produced output: %q", output.String())
			}
		})
	}
}

func TestValidationOutputRejectsInvalidTransformation(t *testing.T) {
	c, err := New(WithValidation(true, true), WithTransform(func(api *model.API) error {
		api.Paths["invalid"] = model.PathItem{}
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: model.NewAPI()})
	c.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
	var output bytes.Buffer
	err = c.Convert(context.Background(), bytes.NewReader(nil), &output, format.FormatOpenAPI, format.FormatOpenAPI)
	if !errors.Is(err, apierrors.ErrValidationFailed) || !strings.Contains(err.Error(), "output") {
		t.Fatalf("Convert() error = %v, want output validation failure", err)
	}
	if output.Len() != 0 {
		t.Fatalf("invalid transformed model produced output: %q", output.String())
	}
}

func TestValidationRejectsDirectParseAndWrite(t *testing.T) {
	api := &model.API{Paths: map[string]model.PathItem{"bad-path": {}}}
	c, err := New(WithValidation(true, true))
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: api})
	c.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
	parsed, err := c.ParseToModel(context.Background(), bytes.NewReader(nil), format.FormatOpenAPI)
	if parsed != nil || !errors.Is(err, apierrors.ErrValidationFailed) || !strings.Contains(err.Error(), "input") {
		t.Fatalf("ParseToModel() = %#v, %v; want nil and input validation error", parsed, err)
	}
	var output bytes.Buffer
	err = c.WriteFromModel(context.Background(), api, &output, format.FormatOpenAPI)
	if !errors.Is(err, apierrors.ErrValidationFailed) || !strings.Contains(err.Error(), "output") || output.Len() != 0 {
		t.Fatalf("WriteFromModel() = %v, output %q; want output validation error and no output", err, output.String())
	}
}

func TestValidateReportsIssuesWhenInputValidationEnabled(t *testing.T) {
	c, err := New(WithValidation(true, false))
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: &model.API{Paths: map[string]model.PathItem{"bad-path": {}}}})
	input := `openapi: 3.0.0
info: {title: Test, version: '1'}
paths: {}
`
	issues, err := c.Validate(context.Background(), strings.NewReader(input), format.FormatOpenAPI)
	if err != nil || len(issues) != 1 || issues[0].Path != "bad-path" {
		t.Fatalf("Validate() = %#v, %v; want one path issue and no error", issues, err)
	}
}

func TestValidatePreservesSchemaAndModelIssuesWithInputValidation(t *testing.T) {
	c, err := New(WithValidation(true, false))
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: &model.API{Paths: map[string]model.PathItem{"bad-path": {}}}})
	input := `openapi: 3.0.0
info: {title: Test}
paths: {}
`
	issues, err := c.Validate(context.Background(), strings.NewReader(input), format.FormatOpenAPI)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	var foundPath, foundSchema bool
	for _, issue := range issues {
		if issue.Path == "bad-path" {
			foundPath = true
		} else {
			foundSchema = true
		}
	}
	if !foundPath || !foundSchema {
		t.Fatalf("Validate() issues = %#v, want schema and model issues", issues)
	}
}

func (m *mockParser) Parse(ctx context.Context, r io.Reader) (*model.API, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.api != nil && m.api.Version == "" {
		m.api.Version = "3.1.0"
	}
	return m.api, nil
}

func (m *mockParser) Format() format.Format {
	return m.fmt
}

func (m *mockParser) SupportsVersion(version string) bool {
	return true
}

type mockWriter struct {
	fmt format.Format
	err error
}

type warningRule struct{}

func (warningRule) Name() string                  { return "test-warning" }
func (warningRule) Level() format.ValidationLevel { return format.LevelWarning }
func (warningRule) Validate(context.Context, *model.API) []format.ValidationError {
	return []format.ValidationError{{Level: format.LevelWarning, Path: "warning", Message: "warning only"}}
}

func TestValidationWarningsDoNotFailConversion(t *testing.T) {
	c, err := New(WithValidation(true, true))
	if err != nil {
		t.Fatal(err)
	}
	c.validator.AddRule(warningRule{})
	c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: model.NewAPI()})
	c.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
	var output bytes.Buffer
	if err := c.Convert(context.Background(), bytes.NewReader(nil), &output, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
		t.Fatalf("warning failed conversion: %v", err)
	}
	if output.String() != "mock output" {
		t.Fatalf("output = %q, want mock output", output.String())
	}
}

func (m *mockWriter) Write(ctx context.Context, api *model.API, w io.Writer) error {
	if m.err != nil {
		return m.err
	}
	_, err := w.Write([]byte("mock output"))
	return err
}

func (m *mockWriter) Format() format.Format {
	return m.fmt
}

func (m *mockWriter) Version() string {
	return "3.1.0"
}

// --- Tests ---

func TestNew(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.parsers == nil || c.writers == nil {
		t.Error("New() did not initialize maps")
	}
	if c.validator == nil {
		t.Error("New() did not initialize validator")
	}
}

func TestRegistration(t *testing.T) {
	c, _ := New()

	// Test Parser Registration
	mp := &mockParser{fmt: format.FormatOpenAPI}
	c.RegisterParser(mp)
	if _, ok := c.parsers[format.FormatOpenAPI]; !ok {
		t.Error("RegisterParser failed to register parser")
	}

	// Test Writer Registration
	mw := &mockWriter{fmt: format.Format("custom")}
	c.RegisterWriter(mw)
	if _, ok := c.writers[format.Format("custom")]; !ok {
		t.Error("RegisterWriter failed to register writer")
	}
}

func TestConvert(t *testing.T) {
	mockAPI := &model.API{Info: model.Info{Title: "Test API"}}

	tests := []struct {
		name       string
		from       format.Format
		to         format.Format
		setup      func(*Converter)
		wantErr    bool
		wantOutput string
	}{
		{
			name: "Success",
			from: format.FormatOpenAPI,
			to:   format.FormatOpenAPI,
			setup: func(c *Converter) {
				c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: mockAPI})
				c.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
			},
			wantErr:    false,
			wantOutput: "mock output",
		},
		{
			name: "Parser Not Registered",
			from: format.FormatOpenAPI,
			to:   format.Format("custom"),
			setup: func(c *Converter) {
				// No parser registered
				c.RegisterWriter(&mockWriter{fmt: format.Format("custom")})
			},
			wantErr: true,
		},
		{
			name: "Writer Not Registered",
			from: format.FormatOpenAPI,
			to:   format.Format("custom"),
			setup: func(c *Converter) {
				c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: mockAPI})
				// No writer registered
			},
			wantErr: true,
		},
		{
			name: "Parser Error",
			from: format.FormatOpenAPI,
			to:   format.Format("custom"),
			setup: func(c *Converter) {
				c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, err: errors.New("parse error")})
				c.RegisterWriter(&mockWriter{fmt: format.Format("custom")})
			},
			wantErr: true,
		},
		{
			name: "Writer Error",
			from: format.FormatOpenAPI,
			to:   format.FormatOpenAPI,
			setup: func(c *Converter) {
				c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: mockAPI})
				c.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI, err: errors.New("write error")})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New()
			if tt.setup != nil {
				tt.setup(c)
			}

			var buf bytes.Buffer
			err := c.Convert(context.Background(), bytes.NewReader([]byte("dummy input")), &buf, tt.from, tt.to)

			if (err != nil) != tt.wantErr {
				t.Errorf("Convert() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && buf.String() != tt.wantOutput {
				t.Errorf("Convert() output = %q, want %q", buf.String(), tt.wantOutput)
			}
		})
	}
}

func TestTransformOption(t *testing.T) {
	// Verify that the WithTransform option works
	wasCalled := false
	transformFn := func(api *model.API) error {
		wasCalled = true
		return nil
	}

	c, _ := New(WithTransform(transformFn))
	c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: &model.API{}})
	c.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})

	err := c.Convert(context.Background(), bytes.NewReader(nil), &bytes.Buffer{}, format.FormatOpenAPI, format.FormatOpenAPI)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	if !wasCalled {
		t.Error("Transform function was not called")
	}
}

func TestCustomPluginFormatsAreNotRestrictedByMigrationPolicy(t *testing.T) {
	const source format.Format = "custom-input"
	const target format.Format = "custom-output"
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(&mockParser{fmt: source, api: &model.API{Version: "custom"}})
	c.RegisterWriter(&mockWriter{fmt: target})
	var out bytes.Buffer
	if err := c.Convert(context.Background(), strings.NewReader("input"), &out, source, target); err != nil || out.String() != "mock output" {
		t.Fatalf("custom conversion = %q, %v", out.String(), err)
	}
}

func TestValidateStillChecksOpenAPIPaths(t *testing.T) {
	c, _ := New()
	c.RegisterParser(&mockParser{
		fmt: format.FormatOpenAPI,
		api: &model.API{
			Paths: map[string]model.PathItem{
				"users": {},
			},
		},
	})

	input := []byte(`
openapi: 3.0.0
info:
  title: Example
  version: 1.0.0
paths: {}
`)
	errs, err := c.Validate(context.Background(), bytes.NewReader(input), format.FormatOpenAPI)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("Validate() returned %d errors, want 1", len(errs))
	}
	if errs[0].Message != "Path must start with /" {
		t.Fatalf("Validate() error message = %q, want %q", errs[0].Message, "Path must start with /")
	}
}

func TestValidateHonorsCanceledContextBeforeRead(t *testing.T) {
	c, _ := New()
	c.RegisterParser(&mockParser{
		fmt: format.FormatOpenAPI,
		api: &model.API{},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Validate(ctx, bytes.NewReader([]byte("openapi: 3.0.0")), format.FormatOpenAPI)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Validate() error = %v, want context.Canceled", err)
	}
}

func TestParseToModelValidateInputUsesFormatAwareRules(t *testing.T) {
	c, _ := New(WithValidation(true, false))
	c.RegisterParser(&mockParser{
		fmt: format.Format("custom"),
		api: &model.API{
			Paths: map[string]model.PathItem{
				"/user/signedup": {},
			},
		},
	})

	api, err := c.ParseToModel(context.Background(), bytes.NewReader([]byte("dummy input")), format.Format("custom"))
	if err != nil {
		t.Fatalf("ParseToModel() error = %v", err)
	}
	if api == nil {
		t.Fatal("ParseToModel() returned nil API")
	}
}

func TestWriteFromModelValidateOutputUsesFormatAwareRules(t *testing.T) {
	c, _ := New(WithValidation(false, true))
	c.RegisterWriter(&mockWriter{fmt: format.Format("custom")})

	api := &model.API{
		Paths: map[string]model.PathItem{
			"/user/signedup": {},
		},
	}

	var buf bytes.Buffer
	if err := c.WriteFromModel(context.Background(), api, &buf, format.Format("custom")); err != nil {
		t.Fatalf("WriteFromModel() error = %v", err)
	}
	if buf.String() != "mock output" {
		t.Fatalf("WriteFromModel() output = %q, want %q", buf.String(), "mock output")
	}
}
