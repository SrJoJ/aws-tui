package tui

import (
	"testing"

	"github.com/SrJoJ/aws-tui/pkg/config"
	"github.com/SrJoJ/aws-tui/pkg/provider"
	"github.com/SrJoJ/aws-tui/pkg/providers/ec2"
)

func TestGetAutocompleteSuggestion(t *testing.T) {
	reg := provider.NewRegistry()
	ec2Prov := ec2.NewProvider()
	_ = reg.Register(ec2Prov)

	cfg := config.DefaultConfig()

	app := NewApp(reg, cfg)
	app.cmdInput.SetLabel(":")

	tests := []struct {
		input    string
		expected string
	}{
		{"p", "rofile"},
		{"pr", "ofile"},
		{"profile", "s"},
		{"r", "eg"},
		{"re", "g"},
		{"reg", "ion"},
		{"reg ", "us-east-1"},
		{"reg u", "s-east-1"},
		{"reg us-w", "est-1"},
		{"e", "c2"},
		{"ins", "tance"},
		{"not-a-command", ""},
	}

	for _, tc := range tests {
		got := app.getAutocompleteSuggestion(tc.input)
		if got != tc.expected {
			t.Errorf("for input %q, expected suggestion %q, got %q", tc.input, tc.expected, got)
		}
	}
}

func TestEscapeTviewTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text [with brackets]", "text [[with brackets]"},
		{"nested [brackets [again]]", "nested [[brackets [[again]]"},
	}

	for _, tc := range tests {
		got := escapeTviewTags(tc.input)
		if got != tc.expected {
			t.Errorf("expected escapeTviewTags(%q) = %q, got %q", tc.input, tc.expected, got)
		}
	}
}
