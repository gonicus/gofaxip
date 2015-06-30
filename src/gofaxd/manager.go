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
	"fmt"
)

type manager struct {
	devices []*Device
}

func newManager(nameprefix string, count uint) (*manager, error) {
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
		if d.GetState() == stateReady {
			d.SetBusy(msg, false)
			return d, nil
		}
	}
	return nil, errors.New("No available modem found.")
}
