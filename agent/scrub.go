package agent

import "regexp"

var credentialPattern = regexp.MustCompile(
	`(?i)(token|api[_-]?key|password|secret|authorization)(\s*[:=]\s*)(\S+)`,
)

// bearerPattern matches "Bearer <token>" to scrub the actual token value.
var bearerPattern = regexp.MustCompile(
	`(?i)(Bearer\s+)(\S+)`,
)

// ScrubCredentials redacts credential values in strings like "TOKEN=abcdef"
// or "api_key: sk-proj-abc". The label and separator are preserved; only the
// value is replaced with a redacted form that keeps the first 4 characters.
func ScrubCredentials(s string) string {
	// Scrub Bearer tokens first (e.g. "Bearer eyJhbG...") before the
	// credential pattern replaces the keyword that precedes them.
	s = bearerPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := bearerPattern.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		prefix, token := parts[1], parts[2]
		if len(token) <= 4 {
			return prefix + "****"
		}
		return prefix + token[:4] + "****"
	})
	s = credentialPattern.ReplaceAllStringFunc(s, func(match string) string {
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
	return s
}
