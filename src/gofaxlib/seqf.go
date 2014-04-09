package gofaxlib

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	SEQ_FILE_NAME = "seqf"
)

func GetSeqFor(subdir string) (seq uint64, err error) {
	seqfname := filepath.Join(Config.Hylafax.Spooldir, subdir, SEQ_FILE_NAME)

	seqf, err := os.OpenFile(seqfname, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	defer seqf.Close()

	if err = syscall.Flock(int(seqf.Fd()), syscall.LOCK_EX); err != nil {
		return
	}

	if _, err = fmt.Fscan(seqf, &seq); err != nil && err.Error() != "EOF" {
		return
	}

	seq += 1

	if err = seqf.Truncate(0); err != nil {
		return
	}
	if _, err = seqf.Seek(0, 0); err != nil {
		return
	}

	if _, err = fmt.Fprintf(seqf, "%d\n", seq); err != nil {
		return
	}

	return
}
