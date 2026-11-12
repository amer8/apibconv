package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/amer8/apibconv/pkg/format"
)

type reportIssue struct {
	Level   format.ValidationLevel `json:"level"`
	Path    string                 `json:"path"`
	Message string                 `json:"message"`
}

type validationReport struct {
	Valid  bool          `json:"valid"`
	Issues []reportIssue `json:"issues"`
	Error  string        `json:"error,omitempty"`
}

// makeValidationReport keeps diagnostic issues even when parsing also failed.
func makeValidationReport(issues []format.ValidationError, validationErr error) validationReport {
	report := validationReport{Valid: validationErr == nil, Issues: make([]reportIssue, 0, len(issues))}
	for _, issue := range issues {
		report.Issues = append(report.Issues, reportIssue{Level: issue.Level, Path: issue.Path, Message: issue.Message})
		if issue.Level == format.LevelError {
			report.Valid = false
		}
	}
	if validationErr != nil {
		report.Error = validationErr.Error()
	}
	return report
}

func writeTextIssues(w io.Writer, issues []reportIssue) error {
	for _, issue := range issues {
		if _, err := fmt.Fprintf(w, "%s: %s: %s\n", issue.Level, issue.Path, issue.Message); err != nil {
			return err
		}
	}
	return nil
}

func writeValidationReport(report validationReport, encoding string, stdout, stderr io.Writer) error {
	if encoding == "json" {
		return json.NewEncoder(stdout).Encode(report)
	}
	if err := writeTextIssues(stderr, report.Issues); err != nil {
		return err
	}
	if report.Valid {
		_, err := fmt.Fprintln(stdout, "Validation successful")
		return err
	}
	return nil
}
