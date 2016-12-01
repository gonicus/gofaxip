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
	"github.com/fiorix/go-eventsocket/eventsocket"
)

// EventStream is a stream of FreeSWITCH Event Socket events
// provided through channels.
type EventStream interface {
	Events() <-chan *eventsocket.Event
	Errors() <-chan error
	Close()
}

type evs struct {
	conn   *eventsocket.Connection
	events chan *eventsocket.Event
	errors chan error
}

// NewEventStream creates a EventStream that continuously reads events
// from a FreeSWITCH Event Socket connection into channes.
func NewEventStream(conn *eventsocket.Connection) EventStream {
	e := &evs{
		conn:   conn,
		events: make(chan *eventsocket.Event),
		errors: make(chan error),
	}
	go e.loop()
	return e
}

func (e *evs) Events() <-chan *eventsocket.Event {
	return e.events
}

func (e *evs) Errors() <-chan error {
	return e.errors
}

func (e *evs) Close() {
	e.conn.Close()
}

func (e *evs) loop() {
	for {
		ev, err := e.conn.ReadEvent()
		if err != nil {
			e.errors <- err
			break
		} else {
			e.events <- ev
		}
	}
}
