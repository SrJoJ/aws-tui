package awsclient

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GetAWSProfiles lists all AWS profiles found in ~/.aws/credentials and ~/.aws/config.
// If none are found, it returns a default mock list for development.
func GetAWSProfiles() []string {
	profilesMap := make(map[string]bool)

	homeDir, err := os.UserHomeDir()
	if err == nil {
		paths := []string{
			filepath.Join(homeDir, ".aws", "credentials"),
			filepath.Join(homeDir, ".aws", "config"),
		}

		for _, path := range paths {
			if file, err := os.Open(path); err == nil {
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					// Ignore comments or empty lines
					if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
						continue
					}
					if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
						profileName := line[1 : len(line)-1]
						// In ~/.aws/config, profiles can be written as [profile my-profile-name]
						if strings.HasPrefix(profileName, "profile ") {
							profileName = strings.TrimPrefix(profileName, "profile ")
						}
						profileName = strings.TrimSpace(profileName)
						if profileName != "" {
							profilesMap[profileName] = true
						}
					}
				}
				file.Close()
			}
		}
	}

	if len(profilesMap) == 0 {
		// Return mock profiles for development/demonstration
		return []string{"default", "prod-admin", "staging-readOnly", "dev-sandbox"}
	}

	// Convert map to slice
	var profiles []string
	for p := range profilesMap {
		profiles = append(profiles, p)
	}
	return profiles
}
