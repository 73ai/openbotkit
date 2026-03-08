package agent

import "regexp"

var credentialPattern = regexp.MustCompile(
	`(?i)(token|api[_-]?key|password|secret|authorization)(\s*[:=]\s*)(\S+)`,
)

// ScrubCredentials redacts credential values in strings like "TOKEN=abcdef"
// or "api_key: sk-proj-abc". The label and separator are preserved; only the
// value is replaced with a redacted form that keeps the first 4 characters.
func ScrubCredentials(s string) string {
	return credentialPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := credentialPattern.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		label, sep, value := parts[1], parts[2], parts[3]
		if len(value) <= 4 {
			return label + sep + "****"
		}
		return label + sep + value[:4] + "****"
	})
}
