package converter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// --- Mocks ---

type mockParser struct {
	fmt format.Format
	api *model.API
	err error
}

func (m *mockParser) Parse(ctx context.Context, r io.Reader) (*model.API, error) {
	if m.err != nil {
		return nil, m.err
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
	return "1.0"
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
	mw := &mockWriter{fmt: format.FormatAsyncAPI}
	c.RegisterWriter(mw)
	if _, ok := c.writers[format.FormatAsyncAPI]; !ok {
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
			to:   format.FormatAsyncAPI,
			setup: func(c *Converter) {
				c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: mockAPI})
				c.RegisterWriter(&mockWriter{fmt: format.FormatAsyncAPI})
			},
			wantErr:    false,
			wantOutput: "mock output",
		},
		{
			name: "Parser Not Registered",
			from: format.FormatOpenAPI,
			to:   format.FormatAsyncAPI,
			setup: func(c *Converter) {
				// No parser registered
				c.RegisterWriter(&mockWriter{fmt: format.FormatAsyncAPI})
			},
			wantErr: true,
		},
		{
			name: "Writer Not Registered",
			from: format.FormatOpenAPI,
			to:   format.FormatAsyncAPI,
			setup: func(c *Converter) {
				c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: mockAPI})
				// No writer registered
			},
			wantErr: true,
		},
		{
			name: "Parser Error",
			from: format.FormatOpenAPI,
			to:   format.FormatAsyncAPI,
			setup: func(c *Converter) {
				c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, err: errors.New("parse error")})
				c.RegisterWriter(&mockWriter{fmt: format.FormatAsyncAPI})
			},
			wantErr: true,
		},
		{
			name: "Writer Error",
			from: format.FormatOpenAPI,
			to:   format.FormatAsyncAPI,
			setup: func(c *Converter) {
				c.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: mockAPI})
				c.RegisterWriter(&mockWriter{fmt: format.FormatAsyncAPI, err: errors.New("write error")})
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
	c.RegisterWriter(&mockWriter{fmt: format.FormatAsyncAPI})

	err := c.Convert(context.Background(), bytes.NewReader(nil), &bytes.Buffer{}, format.FormatOpenAPI, format.FormatAsyncAPI)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	if !wasCalled {
		t.Error("Transform function was not called")
	}
}

func TestValidateSkipsAsyncAPIPathRule(t *testing.T) {
	c, _ := New()
	c.RegisterParser(&mockParser{
		fmt: format.FormatAsyncAPI,
		api: &model.API{
			Paths: map[string]model.PathItem{
				"user/signedup": {},
			},
		},
	})

	errs, err := c.Validate(context.Background(), bytes.NewReader([]byte("dummy input")), format.FormatAsyncAPI)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("Validate() returned unexpected errors: %#v", errs)
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

	errs, err := c.Validate(context.Background(), bytes.NewReader([]byte("dummy input")), format.FormatOpenAPI)
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

func TestParseToModelValidateInputUsesFormatAwareRules(t *testing.T) {
	c, _ := New(WithValidation(true, false))
	c.RegisterParser(&mockParser{
		fmt: format.FormatAsyncAPI,
		api: &model.API{
			Paths: map[string]model.PathItem{
				"user/signedup": {},
			},
		},
	})

	api, err := c.ParseToModel(context.Background(), bytes.NewReader([]byte("dummy input")), format.FormatAsyncAPI)
	if err != nil {
		t.Fatalf("ParseToModel() error = %v", err)
	}
	if api == nil {
		t.Fatal("ParseToModel() returned nil API")
	}
}

func TestWriteFromModelValidateOutputUsesFormatAwareRules(t *testing.T) {
	c, _ := New(WithValidation(false, true))
	c.RegisterWriter(&mockWriter{fmt: format.FormatAsyncAPI})

	api := &model.API{
		Paths: map[string]model.PathItem{
			"user/signedup": {},
		},
	}

	var buf bytes.Buffer
	if err := c.WriteFromModel(context.Background(), api, &buf, format.FormatAsyncAPI); err != nil {
		t.Fatalf("WriteFromModel() error = %v", err)
	}
	if buf.String() != "mock output" {
		t.Fatalf("WriteFromModel() output = %q, want %q", buf.String(), "mock output")
	}
}
