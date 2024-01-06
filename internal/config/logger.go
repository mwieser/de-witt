package config

import "github.com/rs/zerolog"

type Logger struct {
	Level              zerolog.Level
	PrettyPrintConsole bool
}
