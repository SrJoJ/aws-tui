package awsclient

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAWSProfiles(t *testing.T) {
	// Set up mock HOME directory
	tmpHome, err := os.MkdirTemp("", "aws_home_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	awsDir := filepath.Join(tmpHome, ".aws")
	if err := os.Mkdir(awsDir, 0700); err != nil {
		t.Fatal(err)
	}

	// 1. Create a dummy credentials file
	credsContent := `
[default]
aws_access_key_id = test-id
aws_secret_access_key = test-secret

[prod-admin]
aws_access_key_id = prod-id
`
	if err := os.WriteFile(filepath.Join(awsDir, "credentials"), []byte(credsContent), 0600); err != nil {
		t.Fatal(err)
	}

	// 2. Create a dummy config file
	configContent := `
[profile staging-readOnly]
region = us-west-2

[profile dev-sandbox]
region = us-east-1
`
	if err := os.WriteFile(filepath.Join(awsDir, "config"), []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Temporarily override HOME environment variables
	origHome := os.Getenv("HOME")
	origUserprofile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserprofile)
	}()

	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	profiles := GetAWSProfiles()

	expectedProfiles := map[string]bool{
		"default":          true,
		"prod-admin":       true,
		"staging-readOnly": true,
		"dev-sandbox":      true,
	}

	if len(profiles) != len(expectedProfiles) {
		t.Errorf("expected %d profiles, got %d: %v", len(expectedProfiles), len(profiles), profiles)
	}

	for _, p := range profiles {
		if !expectedProfiles[p] {
			t.Errorf("unexpected profile parsed: %s", p)
		}
	}
}

func TestGetAWSProfilesFallback(t *testing.T) {
	// Override home to empty temp dir with no files
	tmpHome, err := os.MkdirTemp("", "aws_home_test_empty_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	origUserprofile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserprofile)
	}()

	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	profiles := GetAWSProfiles()

	// Should fallback to default mock profiles
	if len(profiles) == 0 {
		t.Error("expected fallback profiles, got 0")
	}

	foundDefault := false
	for _, p := range profiles {
		if p == "default" {
			foundDefault = true
			break
		}
	}
	if !foundDefault {
		t.Errorf("expected 'default' to be in fallback profiles, got: %v", profiles)
	}
}
