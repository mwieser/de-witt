package config

type Config struct {
	AppConfig AppConfig `yaml:"de-witt"`
	Logger    Logger
}
