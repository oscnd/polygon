package util

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ToTitleCase(s string) string {
	// * replace underscores with spaces and convert to title case, then remove spaces
	parts := strings.Split(s, "_")
	caser := cases.Title(language.English)
	for i, part := range parts {
		if part != "" {
			parts[i] = caser.String(strings.ToLower(part))
		}
	}
	return strings.Join(parts, "")
}

func ToCamelCase(s string) string {
	// * if string contains underscores, treat as snake case
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		caser := cases.Title(language.English)
		for i, part := range parts {
			if part != "" {
				if i == 0 {
					parts[i] = strings.ToLower(part)
				} else {
					parts[i] = caser.String(strings.ToLower(part))
				}
			}
		}
		return strings.Join(parts, "")
	}

	// * check if string is already camelCase or PascalCase
	if s == "" {
		return s
	}

	// * check if all characters are lowercase (already camelCase)
	if s == strings.ToLower(s) {
		return s
	}

	// * check if all characters are uppercase
	if s == strings.ToUpper(s) {
		return strings.ToLower(s)
	}

	// * treat as PascalCase or mixed case - convert to camelCase
	// * keep first character lowercase, preserve rest
	return strings.ToLower(s[:1]) + s[1:]
}

func ToSingularTitleCase(s string) string {
	// * convert table name to singular title case
	singular := ToSingular(s)
	return ToTitleCase(singular)
}

func ToSingular(s string) string {
	// * simple plural to singular conversion
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "ies") {
		return strings.TrimSuffix(s, "ies") + "y"
	}
	if strings.HasSuffix(s, "ves") {
		return strings.TrimSuffix(s, "ves") + "f"
	}

	// * handle words ending with 's' but not words that naturally end with 's'
	if strings.HasSuffix(s, "s") && len(s) > 1 {
		// * don't strip 's' from words that naturally end with 's'
		naturalEndsWithS := []string{"status", "process", "address", "class", "series"}
		for _, word := range naturalEndsWithS {
			if s == word {
				return s
			}
		}
		// * special case for words ending in 'us' (like 'status' -> 'status')
		if strings.HasSuffix(s, "us") {
			return s
		}
		return strings.TrimSuffix(s, "s")
	}
	return s
}
