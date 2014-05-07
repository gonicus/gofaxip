package main

import (
	"errors"
	"fmt"
)

type manager struct {
	devices []*Device
}

func NewManager(nameprefix string, count uint) (*manager, error) {
	var err error

	m := &manager{
		devices: make([]*Device, count),
	}

	for i := uint(0); i < count; i++ {
		m.devices[i], err = NewDevice(fmt.Sprintf("%v%v", nameprefix, i))
		if err != nil {
			m.SetAllDown()
			return nil, err
		}
	}

	return m, nil
}

func (m *manager) SetAllDown() {
	for _, d := range m.devices {
		if d != nil {
			d.SetDown()
		}
	}
}

func (m *manager) FindDevice(msg string) (*Device, error) {
	for _, d := range m.devices {
		if d.GetState() == STATE_READY {
			d.SetBusy(msg, false)
			return d, nil
		}
	}
	return nil, errors.New("No available modem found.")
}
