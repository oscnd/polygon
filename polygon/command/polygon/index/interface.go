package index

type App interface {
	Verbose() *bool
	Directory() *string
	Config() *Config
}
