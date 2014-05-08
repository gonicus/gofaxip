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

package main

import (
	"errors"
	"gofaxlib"
	"gofaxlib/logger"
	"os"
	"path/filepath"
	"syscall"
)

const (
	STATE_READY = iota
	STATE_BUSY
	STATE_DOWN
	STATE_LOCKED

	FIFO_PREFIX = "FIFO."
	STATUS_DIR  = "status"
)

type Device struct {
	Name       string
	fifoname   string
	statusfile string
	fifostream gofaxlib.FifoStream

	stateSet chan uint
	stateGet chan uint
	errors   chan error
}

func NewDevice(name string) (*Device, error) {
	var err error

	d := Device{
		Name: name,

		fifoname:   filepath.Join(gofaxlib.Config.Hylafax.Spooldir, FIFO_PREFIX+name),
		statusfile: filepath.Join(gofaxlib.Config.Hylafax.Spooldir, STATUS_DIR, name),

		stateSet: make(chan uint),
		stateGet: make(chan uint),
		errors:   make(chan error),
	}

	// Create device FIFO
	stat, err := os.Stat(d.fifoname)
	if err != nil {
		if os.IsNotExist(err) {
			if err = syscall.Mkfifo(d.fifoname, 0600); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		if stat.Mode()&os.ModeNamedPipe == 0 {
			return nil, errors.New("File exists and is not a FIFO")
		}
	}

	go d.stateLoop(STATE_READY)

	d.fifostream = gofaxlib.NewFifoStream(d.fifoname)
	go d.fifoLoop()

	d.SetReady()
	return &d, nil
}

func (d *Device) fifoLoop() {

	for {
		select {
		case msg := <-d.fifostream.Messages():

			if len(msg) == 0 {
				continue
			}
			logger.Logger.Println(d.fifoname, "received message:", msg)

			switch msg[0] {
			case 'H': // Hello
				d.SetReady()
			case 'L': // Lock
				d.SetLocked()
			case 'S': // Set state
				if len(msg) < 2 {
					continue
				}
				switch msg[1] {
				case 'R':
					d.SetReady()
				case 'B':
					d.SetBusy("Sending facsimile", true)
				}
			default:
				logger.Logger.Println("Unhandled message:", msg)
			}

		case err := <-d.fifostream.Errors():
			logger.Logger.Printf("Error in stream for FIFO %s: %v", d.fifoname, err)
			return
		}
	}

}

func (d *Device) stateLoop(state uint) {
	for {
		select {
		case state = <-d.stateSet:
		case d.stateGet <- state:
		}
	}
}

func (d *Device) handle(msg string) {
}

func (d *Device) WriteStatusFile(msg string) {
	sfh, err := os.OpenFile(d.statusfile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		logger.Logger.Print(err)
		return
	}

	if err = syscall.Flock(int(sfh.Fd()), syscall.LOCK_EX); err != nil {
		sfh.Close()
		logger.Logger.Print(err)
		return
	}

	// Truncate after acquiring lock!
	sfh.Truncate(0)

	if _, err := sfh.WriteString(msg); err != nil {
		logger.Logger.Print(err)
		return
	}

	if err = sfh.Close(); err != nil {
		logger.Logger.Print(err)
		return
	}
}

func (d *Device) GetState() uint {
	return <-d.stateGet
}

func (d *Device) SetReady() {
	logger.Logger.Printf("Changing state of modem %v to READY", d.Name)
	d.stateSet <- STATE_READY
	gofaxlib.Faxq.ModemStatus(d.Name, "N")
	d.WriteStatusFile("Running and idle")
	gofaxlib.Faxq.ModemStatusReady(d.Name)
}

func (d *Device) SetBusy(msg string, outbound bool) {
	logger.Logger.Printf("Changing state of modem %v to BUSY", d.Name)
	d.stateSet <- STATE_BUSY

	if outbound {
		gofaxlib.Faxq.ModemStatus(d.Name, "U")
	} else {
		gofaxlib.Faxq.ModemStatus(d.Name, "B")
	}

	if msg == "" {
		msg = "Busy"
	}
	d.WriteStatusFile(msg)
}

func (d *Device) SetDown() {
	logger.Logger.Printf("Changing state of modem %v to DOWN", d.Name)
	d.stateSet <- STATE_DOWN
	gofaxlib.Faxq.ModemStatus(d.Name, "D")
	d.WriteStatusFile("Down")
}

func (d *Device) SetLocked() {
	logger.Logger.Printf("Changing state of modem %v to LOCKED", d.Name)
	d.stateSet <- STATE_LOCKED
	d.WriteStatusFile("Locked for sending")
}
