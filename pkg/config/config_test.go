package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("expected default config to not be nil")
	}

	if cfg.Shortcuts.Global["command_palette"] != ":" {
		t.Errorf("expected command_palette shortcut to be ':', got %s", cfg.Shortcuts.Global["command_palette"])
	}

	if cfg.Providers["ec2"].Type != "builtin" {
		t.Errorf("expected ec2 provider type to be 'builtin', got %s", cfg.Providers["ec2"].Type)
	}
}

func TestReadConfigFile(t *testing.T) {
	content := `
shortcuts:
  global:
    command_palette: ";"
    quit: "x"
    profile_select: "ctrl-s"
  providers:
    ec2:
      "SSH Connect": "c"
providers:
  ec2:
    type: "plugin"
    path: "/bin/ec2-plugin"
`
	tmpfile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := readConfigFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	if cfg.Shortcuts.Global["command_palette"] != ";" {
		t.Errorf("expected command_palette ';', got %s", cfg.Shortcuts.Global["command_palette"])
	}
	if cfg.Shortcuts.Global["quit"] != "x" {
		t.Errorf("expected quit 'x', got %s", cfg.Shortcuts.Global["quit"])
	}
	if cfg.Shortcuts.Global["profile_select"] != "ctrl-s" {
		t.Errorf("expected profile_select 'ctrl-s', got %s", cfg.Shortcuts.Global["profile_select"])
	}
	if cfg.Shortcuts.Providers["ec2"]["SSH Connect"] != "c" {
		t.Errorf("expected ec2 SSH Connect 'c', got %s", cfg.Shortcuts.Providers["ec2"]["SSH Connect"])
	}
	if cfg.Providers["ec2"].Type != "plugin" {
		t.Errorf("expected ec2 type 'plugin', got %s", cfg.Providers["ec2"].Type)
	}
	if cfg.Providers["ec2"].Path != "/bin/ec2-plugin" {
		t.Errorf("expected ec2 path '/bin/ec2-plugin', got %s", cfg.Providers["ec2"].Path)
	}
}
