package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// GenerateTaskID generates a unique task ID in the format: YYYYmmdd-HHMMSS-<6random>
func GenerateTaskID() (string, error) {
	now := time.Now()

	// Format: YYYYmmdd-HHMMSS
	timestamp := now.Format("20060102-150405")

	// Generate 3 random bytes (6 hex chars)
	randomBytes := make([]byte, 3)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randomHex := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("%s-%s", timestamp, randomHex), nil
}

// GenerateBranchName generates a branch name for a task
// Format: <prefix>/<agent>/<id>
func GenerateBranchName(prefix, agent, taskID string) string {
	// Sanitize agent name (remove invalid characters)
	agent = SanitizeName(agent)
	return fmt.Sprintf("%s/%s/%s", prefix, agent, taskID)
}

// SanitizeName sanitizes a name for use in Git branches
// Removes shell metacharacters and invalid Git ref characters
func SanitizeName(name string) string {
	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")

	// Remove or replace invalid characters
	// Git refs cannot contain: ~, ^, :, ?, *, [, \, .., @{, //
	// Shell metacharacters: $, `, &, |, ;, <, >, (, ), {, }
	invalidChars := []string{
		"~", "^", ":", "?", "*", "[", "\\", "..",
		"$", "`", "&", "|", ";", "<", ">", "(", ")", "{", "}",
		"@{", "//",
	}

	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, char, "")
	}

	// Convert to lowercase for consistency
	name = strings.ToLower(name)

	// Remove leading/trailing hyphens or slashes
	name = strings.Trim(name, "-/")

	return name
}

// ValidateTaskID validates a task ID format
// Task IDs can be any non-empty string that is safe for use in filenames and Git branch names
func ValidateTaskID(id string) bool {
	// Must not be empty
	if id == "" {
		return false
	}

	// Must not be too long (reasonable limit for filenames and branch names)
	if len(id) > 255 {
		return false
	}

	// Check for invalid characters that would cause issues in:
	// - Filenames: / \ : * ? " < > |
	// - Git refs: ~, ^, :, ?, *, [, \, .., @{, //, leading/trailing dots or slashes
	// - Shell safety: $, `, &, |, ;, <, >, (, ), {, }, newlines, tabs
	invalidChars := []string{
		"/", "\\", ":", "*", "?", "\"", "<", ">", "|",
		"~", "^", "[", "..",
		"$", "`", "&", ";", "(", ")", "{", "}",
		"\n", "\r", "\t",
		"@{", "//",
	}

	for _, char := range invalidChars {
		if strings.Contains(id, char) {
			return false
		}
	}

	// Cannot start or end with dots or spaces (problematic for filesystems)
	if strings.HasPrefix(id, ".") || strings.HasSuffix(id, ".") {
		return false
	}
	if strings.HasPrefix(id, " ") || strings.HasSuffix(id, " ") {
		return false
	}

	return true
}
