package logger

import (
	"log"
	"log/syslog"
)

const (
	LOG_PRIORITY = syslog.LOG_DAEMON | syslog.LOG_INFO
	LOG_FLAGS    = log.Lshortfile
)

var (
	Logger *log.Logger
)

func init() {
	var err error
	log.SetFlags(LOG_FLAGS)
	Logger, err = syslog.NewLogger(LOG_PRIORITY, LOG_FLAGS)
	if err != nil {
		log.Fatal(err)
	}

}
