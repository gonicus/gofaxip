package gofaxlib

import (
	"github.com/fiorix/go-eventsocket/eventsocket"
)

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
