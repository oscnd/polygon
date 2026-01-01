package common

import (
	"os"

	"github.com/bsthun/gut"
	"gopkg.in/yaml.v3"
)

func Config[T any]() *T {
	// * parse arguments
	path := os.Getenv("BACKEND_CONFIG_PATH")
	if path == "" {
		path = ".local/config.yml"
	}

	// * declare struct
	config := new(T)

	// * read config
	yml, err := os.ReadFile(path)
	if err != nil {
		gut.Fatal("unable to read configuration file", err)
	}

	// * parse config
	if err := yaml.Unmarshal(yml, config); err != nil {
		gut.Fatal("Unable to parse configuration file", err)
	}

	// * validate config
	if err := gut.Validate(config); err != nil {
		gut.Fatal("invalid configuration", err)
	}

	return config
}
