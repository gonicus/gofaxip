package gofaxlib

import (
	"fmt"
	"gofaxlib/logger"
	"path/filepath"
)

const (
	LOG_DIR         = "log"
	COMMID_FORMAT   = "%08d"
	LOG_FILE_FORMAT = "c%s"
)

type SessionLogger interface {
	CommSeq() uint64
	CommId() string
	Logfile() string

	Log(v ...interface{})
}

type hylasessionlog struct {
	commseq uint64
	commid  string

	logfile string
}

func NewSessionLogger() (SessionLogger, error) {
	// Fetch commid and log file name
	commseq, err := GetSeqFor(LOG_DIR)
	if err != nil {
		return nil, err
	}
	commid := fmt.Sprintf(COMMID_FORMAT, commseq)
	logfile := filepath.Join(LOG_DIR, fmt.Sprintf(LOG_FILE_FORMAT, commid))

	l := &hylasessionlog{
		commseq: commseq,
		commid:  commid,
		logfile: logfile,
	}

	return l, nil
}

func (h *hylasessionlog) Log(v ...interface{}) {
	logger.Logger.Println(v...)
	if err := AppendLog(h.logfile, v...); err != nil {
		logger.Logger.Print(err)
	}
}

func (h *hylasessionlog) CommSeq() uint64 {
	return h.commseq
}

func (h *hylasessionlog) CommId() string {
	return h.commid
}

func (h *hylasessionlog) Logfile() string {
	return h.logfile
}
