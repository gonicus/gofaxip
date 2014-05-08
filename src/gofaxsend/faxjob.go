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
	"code.google.com/p/go-uuid/uuid"
)

// Fax job containing everything FreeSWITCH needs
type FaxJob struct {
	// FreeSWITCH Channel UUID (we generate this)
	UUID uuid.UUID
	// Destination number
	Number string
	// Caller ID number
	Cidnum string
	// Caller ID name
	Cidname string
	// TIFF file to send
	Filename string
	// Use ECM (default: true)
	UseECM bool
	// Fax ident
	Ident string
	// String for header (i.e. sender company name)
	// Page header with timestamp, header, ident, pageno will be added
	// if this Header is non empty
	Header string
}

func NewFaxJob() *FaxJob {
	return &FaxJob{
		UUID:   uuid.NewRandom(),
		UseECM: true,
	}
}
