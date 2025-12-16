package sequel

import (
	"sort"
	"strings"
)

func SortedTableKeys(m map[string]*Table) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func SortedFunctionKeys(m map[string]*Function) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func SortedTriggerKeys(m map[string]*Trigger) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func SplitTableItems(content string) []string {
	var items []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range content {
		if char == '(' {
			parenDepth++
			current.WriteRune(char)
		} else if char == ')' {
			parenDepth--
			current.WriteRune(char)
		} else if char == ',' && parenDepth == 0 {
			items = append(items, current.String())
			current.Reset()
		} else {
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		items = append(items, current.String())
	}

	return items
}
