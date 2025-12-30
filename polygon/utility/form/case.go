package form

import (
	"strings"
	"unicode"
)

func CaseParser(s string) []string {
	if s == "" {
		return []string{}
	}

	var segments []string
	var current strings.Builder

	for i, r := range s {
		if i == 0 {
			current.WriteRune(unicode.ToLower(r))
			continue
		}

		// * split on uppercase letters or underscores
		if unicode.IsUpper(r) || r == '_' {
			// * add current segment if not empty
			if current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
			// * skip underscores
			if r != '_' {
				current.WriteRune(unicode.ToLower(r))
			}
		} else {
			current.WriteRune(r)
		}
	}

	// * add last segment
	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

// ToCamelCase converts a string to camel case
func ToCamelCase(s string) string {
	segments := CaseParser(s)
	if len(segments) == 0 {
		return s
	}

	// First segment is lowercase
	result := segments[0]

	// Subsequent segments have first letter uppercase
	for i := 1; i < len(segments); i++ {
		if len(segments[i]) > 0 {
			result += strings.ToUpper(segments[i][:1]) + segments[i][1:]
		}
	}

	return result
}

// ToPascalCase converts a string to pascal case
func ToPascalCase(s string) string {
	segments := CaseParser(s)
	if len(segments) == 0 {
		return s
	}

	var result strings.Builder
	for _, segment := range segments {
		if len(segment) > 0 {
			result.WriteString(strings.ToUpper(segment[:1]))
			result.WriteString(segment[1:])
		}
	}

	return result.String()
}

// ToSnakeCase converts a string to snake case
func ToSnakeCase(s string) string {
	segments := CaseParser(s)
	if len(segments) == 0 {
		return s
	}

	return strings.Join(segments, "_")
}

// ToSnakeCasePlural converts a string to snake case plural form
func ToSnakeCasePlural(s string) string {
	segments := CaseParser(s)
	if len(segments) == 0 {
		return s
	}

	// * make the last segment plural
	if len(segments) > 0 {
		lastSeg := segments[len(segments)-1]
		segments[len(segments)-1] = CasePluralize(lastSeg)
	}

	return strings.Join(segments, "_")
}

// ToSingular converts a string to singular form
func ToSingular(s string) string {
	segments := CaseParser(s)
	if len(segments) == 0 {
		return s
	}

	// Make the last segment singular
	if len(segments) > 0 {
		lastSeg := segments[len(segments)-1]
		segments[len(segments)-1] = CaseSingularize(lastSeg)
	}

	return strings.Join(segments, "_")
}

// ToSingularTitleCase converts a string to singular title case
func ToSingularTitleCase(s string) string {
	singular := ToSingular(s)
	return ToPascalCase(singular)
}

// CasePluralize makes a word plural
func CasePluralize(word string) string {
	if word == "" {
		return word
	}

	lower := strings.ToLower(word)

	// Basic English pluralization rules
	switch {
	case strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "sh") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "x") || strings.HasSuffix(lower, "z"):
		return word + "es"
	case strings.HasSuffix(lower, "y") && !strings.HasSuffix(lower, "ay") && !strings.HasSuffix(lower, "ey") && !strings.HasSuffix(lower, "iy") && !strings.HasSuffix(lower, "oy") && !strings.HasSuffix(lower, "uy"):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(lower, "f"):
		return word[:len(word)-1] + "ves"
	case strings.HasSuffix(lower, "fe"):
		return word[:len(word)-2] + "ves"
	default:
		return word + "s"
	}
}

// CaseSingularize makes a word singular
func CaseSingularize(word string) string {
	if word == "" {
		return word
	}

	lower := strings.ToLower(word)

	// Basic English singularization rules (reverse of pluralize)
	switch {
	case strings.HasSuffix(lower, "ies"):
		return word[:len(word)-3] + "y"
	case strings.HasSuffix(lower, "ves"):
		return word[:len(word)-3] + "f"
	case strings.HasSuffix(lower, "es"):
		// Check if it's one of the special cases
		if strings.HasSuffix(lower, "ses") || strings.HasSuffix(lower, "shes") || strings.HasSuffix(lower, "ches") || strings.HasSuffix(lower, "xes") || strings.HasSuffix(lower, "zes") {
			return word[:len(word)-2]
		}
		fallthrough
	case strings.HasSuffix(lower, "s"):
		// Simple case: just remove 's'
		if len(word) > 1 {
			return word[:len(word)-1]
		}
		return word
	default:
		return word
	}
}
