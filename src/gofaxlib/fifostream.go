package gofaxlib

import (
	"io/ioutil"
	"os"
	"strings"
)

type FifoStream interface {
	Messages() <-chan string
	Errors() <-chan error
}

type fifo struct {
	name     string
	messages chan string
	errors   chan error
}

func NewFifoStream(name string) FifoStream {
	f := &fifo{
		name:     name,
		messages: make(chan string),
		errors:   make(chan error),
	}
	go f.loop()
	return f
}

func (f *fifo) Messages() <-chan string {
	return f.messages
}

func (f *fifo) Errors() <-chan error {
	return f.errors
}

func (f *fifo) loop() {
	for {
		fifofile, err := os.Open(f.name)
		if err != nil {
			f.errors <- err
			return
		}

		// BUG(markus): split at null or \n
		// until EOF
		msg, err := ioutil.ReadAll(fifofile)
		fifofile.Close()
		if err != nil {
			f.errors <- err
			return
		}
		msgstr := strings.TrimSpace(string(msg))
		if msgstr != "" {
			f.messages <- msgstr
		}

	}
}
