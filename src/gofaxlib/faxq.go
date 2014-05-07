package gofaxlib

import (
	"fmt"
	"gofaxlib/logger"
	"path/filepath"
)

const (
	// No polling
	// Class 2.0: (0,1),(0-5),(0-4),(0-2),(0-3),(0-3),(0),(0-7)
	// http://www.hylafax.org/site2/setup-advanced.html
	CAPABILITIES = "pcbffff01"
	FAXQ_FIFO    = "FIFO"
)

var (
	Faxq *faxqfifo
)

type message struct {
	msg string
	err chan error
}

type faxqfifo struct {
	msgchan chan message
}

func init() {
	Faxq = &faxqfifo{
		msgchan: make(chan message),
	}
	go Faxq.messageLoop()
}

func (f *faxqfifo) getFilename() string {
	return filepath.Join(Config.Hylafax.Spooldir, FAXQ_FIFO)
}

func (f *faxqfifo) messageLoop() {
	for {
		m := <-f.msgchan
		err := SendFIFO(f.getFilename(), m.msg)
		if err != nil {
			m.err <- err
		} else {
			m.err <- nil
		}
	}
}

func (f *faxqfifo) Send(msg string) error {
	logger.Logger.Printf("Sending message to %s: %s", f.getFilename(), msg)
	m := message{
		msg: msg,
		err: make(chan error),
	}
	f.msgchan <- m
	return <-m.err
}

func (f *faxqfifo) ModemStatus(modem string, msg string) error {
	return f.Send(fmt.Sprintf("+%s:%s", modem, msg))
}

func (f *faxqfifo) ModemStatusReady(modem string) error {
	return f.ModemStatus(modem, "R"+CAPABILITIES)
}

func (f *faxqfifo) ReceiveStatus(modem string, msg string) error {
	return f.Send(fmt.Sprintf("@%s:%s", modem, msg))
}

func (f *faxqfifo) JobStatus(jobid string, msg string) error {
	return f.Send(fmt.Sprintf("*%s:%s", jobid, msg))
}
