package gofaxlib

import (
	"os"
)

func SendFIFO(filename string, msg string) error {
	fifo, err := os.OpenFile(filename, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer fifo.Close()

	_, err = fifo.WriteString(msg + "\x00")
	if err != nil {
		return err
	}
	return nil
}
