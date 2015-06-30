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
	"os"
)

// SendFIFO sends a FIFO message to given FIFO file name
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
