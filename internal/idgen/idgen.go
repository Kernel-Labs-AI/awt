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
func ValidateTaskID(id string) bool {
	// Format: YYYYmmdd-HHMMSS-<6hex>
	// Example: 20250110-120000-abc123
	if len(id) != 21 {
		return false
	}

	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		return false
	}

	// Check date part (8 digits)
	if len(parts[0]) != 8 {
		return false
	}

	// Check time part (6 digits)
	if len(parts[1]) != 6 {
		return false
	}

	// Check random part (6 hex chars)
	if len(parts[2]) != 6 {
		return false
	}

	return true
}
