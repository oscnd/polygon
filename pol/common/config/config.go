package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func New[T any](directory string) (*T, error) {
	// * construct config file path
	configPath := path.Join(directory, "polygon.yml")

	// * read config file
	bytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read configuration file: %w", err)
	}

	// * process template replacements
	templated, err := Template(bytes)
	if err != nil {
		return nil, fmt.Errorf("error processing templates: %w", err)
	}

	// * create new config instance
	config := new(T)

	// * parse config
	if err := yaml.Unmarshal(templated, config); err != nil {
		return nil, fmt.Errorf("unable to parse configuration file: %w", err)
	}

	return config, nil
}

func Template(bytes []byte) ([]byte, error) {
	// * regex to find braced templates
	templateRegex := regexp.MustCompile(`\{\{\s*([^}]+)\s*}}`)

	processed := templateRegex.ReplaceAllFunc(bytes, func(match []byte) []byte {
		// * extract content inside braces
		content := strings.TrimSpace(string(match[2 : len(match)-2]))

		// * split by separator
		parts := strings.Split(content, "||")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}

		// * check each part
		for _, part := range parts {
			if strings.HasPrefix(part, "env.") {
				key := strings.TrimPrefix(part, "env.")
				value := os.Getenv(key)
				if value != "" {
					return []byte(value)
				}
			} else if part != "" {
				value, err := Nested(part)
				if err != nil {
					return []byte(part)
				}
				return []byte(value)
			}
		}

		// * no valid value found, return empty
		return []byte("")
	})

	return processed, nil
}

func Nested(value string) (string, error) {
	// * try to parse as json
	var result any
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return "", err
	}

	// * convert back to yaml
	bytes, err := yaml.Marshal(result)
	if err != nil {
		return "", err
	}

	// * remove trailing newline
	return strings.TrimSuffix(string(bytes), "\n"), nil
}
