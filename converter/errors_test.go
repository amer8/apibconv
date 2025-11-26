package converter

import (
	"errors"
	"testing"
)

func TestParseError(t *testing.T) {
	cause := errors.New("invalid JSON syntax")
	err := NewParseError("OpenAPI", cause)

	// Test Error() method
	expected := "failed to parse OpenAPI: invalid JSON syntax"
	if err.Error() != expected {
		t.Errorf("ParseError.Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap() method
	if !errors.Is(err, cause) {
		t.Error("ParseError should unwrap to its cause")
	}

	// Test with nil cause
	errNilCause := &ParseError{Format: "AsyncAPI"}
	expectedNil := "failed to parse AsyncAPI"
	if errNilCause.Error() != expectedNil {
		t.Errorf("ParseError.Error() with nil cause = %q, want %q", errNilCause.Error(), expectedNil)
	}
}

func TestConversionError(t *testing.T) {
	cause := errors.New("schema incompatible")
	err := NewConversionError("OpenAPI 3.0", "API Blueprint", cause)

	// Test Error() method
	expected := "failed to convert from OpenAPI 3.0 to API Blueprint: schema incompatible"
	if err.Error() != expected {
		t.Errorf("ConversionError.Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap() method
	if !errors.Is(err, cause) {
		t.Error("ConversionError should unwrap to its cause")
	}

	// Test with nil cause
	errNilCause := &ConversionError{From: "A", To: "B"}
	expectedNil := "failed to convert from A to B"
	if errNilCause.Error() != expectedNil {
		t.Errorf("ConversionError.Error() with nil cause = %q, want %q", errNilCause.Error(), expectedNil)
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("info.title", "title is required")

	// Test Error() method
	expected := "validation error at info.title: title is required"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap() method
	if !errors.Is(err, ErrValidationFailed) {
		t.Error("ValidationError should unwrap to ErrValidationFailed")
	}

	// Test with empty field
	errNoField := &ValidationError{Message: "something is wrong"}
	expectedNoField := "validation error: something is wrong"
	if errNoField.Error() != expectedNoField {
		t.Errorf("ValidationError.Error() with no field = %q, want %q", errNoField.Error(), expectedNoField)
	}

	// Test with cause
	cause := errors.New("underlying error")
	errWithCause := &ValidationError{Field: "test", Message: "msg", Cause: cause}
	if !errors.Is(errWithCause, cause) {
		t.Error("ValidationError with cause should unwrap to cause")
	}
}

func TestValidationErrors(t *testing.T) {
	errs := &ValidationErrors{}

	// Test empty errors
	if errs.HasErrors() {
		t.Error("Empty ValidationErrors should not have errors")
	}
	if errs.Count() != 0 {
		t.Errorf("Empty ValidationErrors.Count() = %d, want 0", errs.Count())
	}

	// Test Error() with no errors
	expected := "validation failed with no specific errors"
	if errs.Error() != expected {
		t.Errorf("Empty ValidationErrors.Error() = %q, want %q", errs.Error(), expected)
	}

	// Add one error
	errs.Add("info.title", "title is required")
	if !errs.HasErrors() {
		t.Error("ValidationErrors with errors should have errors")
	}
	if errs.Count() != 1 {
		t.Errorf("ValidationErrors.Count() = %d, want 1", errs.Count())
	}

	// Test Error() with one error
	expectedOne := "validation error at info.title: title is required"
	if errs.Error() != expectedOne {
		t.Errorf("ValidationErrors.Error() with one error = %q, want %q", errs.Error(), expectedOne)
	}

	// Add more errors
	errs.Add("info.version", "version is required")
	errs.Add("paths", "paths cannot be empty")

	if errs.Count() != 3 {
		t.Errorf("ValidationErrors.Count() = %d, want 3", errs.Count())
	}

	// Test Error() with multiple errors
	if !contains(errs.Error(), "validation failed with 3 errors") {
		t.Errorf("ValidationErrors.Error() should mention 3 errors: %s", errs.Error())
	}

	// Test Unwrap()
	if !errors.Is(errs, ErrValidationFailed) {
		t.Error("ValidationErrors should unwrap to ErrValidationFailed")
	}
}

func TestVersionError(t *testing.T) {
	err := NewVersionError("4.0.0", "OpenAPI 4.0 is not supported")

	// Test Error() method
	expected := "version error for 4.0.0: OpenAPI 4.0 is not supported"
	if err.Error() != expected {
		t.Errorf("VersionError.Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap() method
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Error("VersionError should unwrap to ErrUnsupportedVersion")
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrNilSpec", ErrNilSpec, "specification cannot be nil"},
		{"ErrInvalidJSON", ErrInvalidJSON, "invalid JSON format"},
		{"ErrInvalidYAML", ErrInvalidYAML, "invalid YAML format"},
		{"ErrUnsupportedVersion", ErrUnsupportedVersion, "unsupported specification version"},
		{"ErrUnsupportedFormat", ErrUnsupportedFormat, "unsupported output format"},
		{"ErrInvalidInput", ErrInvalidInput, "invalid input"},
		{"ErrConversionFailed", ErrConversionFailed, "conversion failed"},
		{"ErrValidationFailed", ErrValidationFailed, "validation failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}

func TestErrorsIs(t *testing.T) {
	// Test that wrapped errors work with errors.Is
	parseErr := NewParseError("OpenAPI", ErrInvalidJSON)
	if !errors.Is(parseErr, ErrInvalidJSON) {
		t.Error("ParseError wrapping ErrInvalidJSON should satisfy errors.Is(ErrInvalidJSON)")
	}

	convErr := NewConversionError("A", "B", ErrUnsupportedFormat)
	if !errors.Is(convErr, ErrUnsupportedFormat) {
		t.Error("ConversionError wrapping ErrUnsupportedFormat should satisfy errors.Is(ErrUnsupportedFormat)")
	}
}

func TestErrorsAs(t *testing.T) {
	// Test errors.As with ParseError
	parseErr := NewParseError("OpenAPI", errors.New("test"))
	var pe *ParseError
	if !errors.As(parseErr, &pe) {
		t.Error("errors.As should work with ParseError")
	}
	if pe.Format != "OpenAPI" {
		t.Errorf("ParseError.Format = %q, want %q", pe.Format, "OpenAPI")
	}

	// Test errors.As with ConversionError
	convErr := NewConversionError("A", "B", errors.New("test"))
	var ce *ConversionError
	if !errors.As(convErr, &ce) {
		t.Error("errors.As should work with ConversionError")
	}
	if ce.From != "A" || ce.To != "B" {
		t.Errorf("ConversionError fields incorrect: From=%q, To=%q", ce.From, ce.To)
	}

	// Test errors.As with ValidationError
	valErr := NewValidationError("field", "message")
	var ve *ValidationError
	if !errors.As(valErr, &ve) {
		t.Error("errors.As should work with ValidationError")
	}
	if ve.Field != "field" {
		t.Errorf("ValidationError.Field = %q, want %q", ve.Field, "field")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
