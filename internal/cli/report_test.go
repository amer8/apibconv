package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
)

func TestValidationReportSeverityAndPaths(t *testing.T) {
	issues := make([]format.ValidationError, 0, 3)
	issues = append(issues,
		format.ValidationError{Level: format.LevelWarning, Path: "/info", Message: "review this"},
		format.ValidationError{Level: format.LevelInfo, Path: "/paths", Message: "information"},
	)
	report := makeValidationReport(issues, nil)
	if !report.Valid {
		t.Fatal("warnings and informational findings must not fail validation")
	}
	var stdout, stderr bytes.Buffer
	if err := writeValidationReport(report, "text", &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "warning: /info: review this") || !strings.Contains(stdout.String(), "Validation successful") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	issues = append(issues, format.ValidationError{Level: format.LevelError, Path: "/paths/~1items", Message: "invalid"})
	report = makeValidationReport(issues, errors.New("parse failed"))
	stdout.Reset()
	if err := writeValidationReport(report, "json", &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var got validationReport
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Valid || got.Error != "parse failed" || len(got.Issues) != 3 || got.Issues[2].Path != "/paths/~1items" {
		t.Fatalf("report = %+v", got)
	}
}
