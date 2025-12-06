package index

type Config struct {
	Server *string `yaml:"server" validate:"required"`
}
