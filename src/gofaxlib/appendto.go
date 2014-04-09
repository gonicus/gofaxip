package gofaxlib

import (
	"log"
	"os"
	"syscall"
)

const (
	APPENDLOG_FLAGS = log.Ldate | log.Ltime
)

// Append line to file (adding line break)
func AppendTo(filename string, line string) (err error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return
	}

	if _, err = f.WriteString(line + "\n"); err != nil {
		return
	}
	return
}

// Append log message to file (adding line break and timestamp)
func AppendLog(filename string, v ...interface{}) (err error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return
	}

	l := log.New(f, "", APPENDLOG_FLAGS)
	l.Println(v...)

	return
}
