package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseFlagsInterspersed(t *testing.T) {
	for _, tc := range []struct {
		name, output, encoding string
		args                   []string
		positional             []string
		verbose                bool
	}{
		{"flags around input", "out.json", "json", []string{"--verbose", "in.apib", "-o", "out.json", "-e=json"}, []string{"in.apib"}, true},
		{"stdin and flags", "", "json", []string{"-", "-e", "json"}, []string{"-"}, false},
		{"end of options", "", "", []string{"--verbose", "--", "-input.apib"}, []string{"-input.apib"}, true},
		{"option value starts with dash", "-out.json", "", []string{"in.apib", "-o", "-out.json"}, []string{"in.apib"}, false},
		{"last option wins", "second.json", "", []string{"-o", "first.json", "in.apib", "-o", "second.json"}, []string{"in.apib"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags, args, err := ParseFlags(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(args, tc.positional) || flags.Output != tc.output || flags.Encoding != tc.encoding || flags.Verbose != tc.verbose {
				t.Fatalf("got args=%q flags=%+v", args, flags)
			}
		})
	}
}

func TestParseFlagsRejectsUnknownOrMissingOptionsAfterInput(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"in.apib", "--unknown"}, "flag provided but not defined"},
		{[]string{"in.apib", "--output"}, "flag needs an argument"},
	} {
		_, _, err := ParseFlags(tc.args)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("ParseFlags(%q) = %v, want %q", tc.args, err, tc.want)
		}
	}
}
