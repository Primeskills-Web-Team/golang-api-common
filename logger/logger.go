package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Setup(isDev bool) {
	var l zerolog.Logger
	if isDev {
		l = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Logger()
		l.Level(zerolog.DebugLevel)
	} else {
		l = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
	log.Logger = l
}
