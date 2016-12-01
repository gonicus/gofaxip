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
	"io/ioutil"
	"os"
	"strings"
)

// FifoStream provides a channel of messages received on a FIFO
type FifoStream interface {
	Messages() <-chan string
	Errors() <-chan error
}

type fifo struct {
	name     string
	messages chan string
	errors   chan error
}

// NewFifoStream creates a FifoStream reading from a FIFO with given filename
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
