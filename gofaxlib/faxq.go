// This file is part of the GOfax.IP project - https://github.com/gonicus/gofaxip
// Copyright (C) 2014 GONICUS GmbH, Germany - http://www.gonicus.de
//
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; version 2
// of the License.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

package gofaxlib

import (
	"fmt"
	"path/filepath"

	"github.com/gonicus/gofaxip/gofaxlib/logger"
)

const (
	// No polling
	// Class 2.0: (0,1),(0-5),(0-4),(0-2),(0-3),(0-3),(0),(0-7)
	// http://www.hylafax.org/site2/setup-advanced.html
	capabilities = "pcbffff01"
	faxqFifoName = "FIFO"
)

var (
	// Faxq provides functionalty to send notification messages to faxq's FIFO
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
	return filepath.Join(Config.Hylafax.Spooldir, faxqFifoName)
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
	return f.ModemStatus(modem, "R"+capabilities)
}

func (f *faxqfifo) ReceiveStatus(modem string, msg string) error {
	return f.Send(fmt.Sprintf("@%s:%s", modem, msg))
}

func (f *faxqfifo) JobStatus(jobid string, msg string) error {
	return f.Send(fmt.Sprintf("*%s:%s", jobid, msg))
}
