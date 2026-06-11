package utils

import "strings"

// NormalizePhone converts any Indian phone number format to the canonical
// 11-digit local format (0XXXXXXXXXX) used by Exotel's API responses.
//
// Accepted input formats → output:
//   07948224931   → 07948224931  (already canonical)
//   7948224931    → 07948224931  (10-digit, prepend 0)
//   917948224931  → 07948224931  (12-digit with country code)
//   +917948224931 → 07948224931  (E.164)
//   0917948224931 → 07948224931  (13-digit)
//
// Non-Indian or unrecognised lengths are returned stripped of non-digits only.
func NormalizePhone(s string) string {
	// Strip everything except digits.
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()

	switch len(digits) {
	case 10: // 7948224931 → 07948224931
		return "0" + digits
	case 11: // 07948224931 — already canonical
		return digits
	case 12: // 917948224931 → 07948224931
		if strings.HasPrefix(digits, "91") {
			return "0" + digits[2:]
		}
	case 13: // 0917948224931 → 07948224931
		if strings.HasPrefix(digits, "091") {
			return "0" + digits[3:]
		}
	}
	return digits
}
