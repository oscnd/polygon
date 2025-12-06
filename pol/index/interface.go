package index

type App interface {
	Directory() *string
	Config() *Config
}
