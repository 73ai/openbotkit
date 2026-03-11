package contacts

import (
	"net/mail"
	"strings"
)

func NormalizePhone(raw string) string {
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	s := digits.String()
	if s == "" {
		return ""
	}
	return "+" + s
}

func NormalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func ParseEmailAddr(raw string) (name, email string) {
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		trimmed := strings.TrimSpace(raw)
		if strings.Contains(trimmed, "@") {
			return "", NormalizeEmail(trimmed)
		}
		return "", ""
	}
	return addr.Name, NormalizeEmail(addr.Address)
}

func ExtractPhoneFromJID(jid string) string {
	digits, _, ok := strings.Cut(jid, "@")
	if !ok || digits == "" {
		return ""
	}
	return "+" + digits
}
