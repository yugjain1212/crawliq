package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/yugjain1212/crawliq/config"
)

func Init(cfg config.LogConfig) {
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	zerolog.TimeFieldFormat = time.RFC3339

	var output zerolog.ConsoleWriter
	if cfg.Pretty {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()
		return
	}
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

}

func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		log.Warn().Str("ConfiguredLevel", level).Msg("Unrecognized log Level, defaulting to INFO")
		return zerolog.InfoLevel
	}
}
