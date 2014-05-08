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
	"gofaxlib/logger"
	"path/filepath"
)

const (
	LOG_DIR         = "log"
	COMMID_FORMAT   = "%08d"
	LOG_FILE_FORMAT = "c%s"
)

type SessionLogger interface {
	CommSeq() uint64
	CommId() string
	Logfile() string

	Log(v ...interface{})
}

type hylasessionlog struct {
	commseq uint64
	commid  string

	logfile string
}

func NewSessionLogger() (SessionLogger, error) {
	// Fetch commid and log file name
	commseq, err := GetSeqFor(LOG_DIR)
	if err != nil {
		return nil, err
	}
	commid := fmt.Sprintf(COMMID_FORMAT, commseq)
	logfile := filepath.Join(LOG_DIR, fmt.Sprintf(LOG_FILE_FORMAT, commid))

	l := &hylasessionlog{
		commseq: commseq,
		commid:  commid,
		logfile: logfile,
	}

	return l, nil
}

func (h *hylasessionlog) Log(v ...interface{}) {
	logger.Logger.Println(v...)
	if err := AppendLog(h.logfile, v...); err != nil {
		logger.Logger.Print(err)
	}
}

func (h *hylasessionlog) CommSeq() uint64 {
	return h.commseq
}

func (h *hylasessionlog) CommId() string {
	return h.commid
}

func (h *hylasessionlog) Logfile() string {
	return h.logfile
}
