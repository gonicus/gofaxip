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
	logDir        = "log"
	commIDFormat  = "%08d"
	logFileFormat = "c%s"
)

// SessionLogger is a logger that logs messages both to
// the processes log facility and a HylaFax session log file
type SessionLogger interface {
	CommSeq() uint64
	CommID() string
	Logfile() string

	Log(v ...interface{})
	Logf(format string, v ...interface{})
}

type hylasessionlog struct {
	jobid   uint
	commseq uint64
	commid  string

	logfile string
}

// NewSessionLogger assigns a CommID and opens a session log file
func NewSessionLogger(jobid uint) (SessionLogger, error) {
	// Fetch commid and log file name
	commseq, err := GetSeqFor(logDir)
	if err != nil {
		return nil, err
	}
	commid := fmt.Sprintf(commIDFormat, commseq)
	logfile := filepath.Join(logDir, fmt.Sprintf(logFileFormat, commid))

	l := &hylasessionlog{
		jobid:   jobid,
		commseq: commseq,
		commid:  commid,
		logfile: logfile,
	}

	return l, nil
}

func (h *hylasessionlog) Log(v ...interface{}) {
	if h.jobid != 0 {
		logger.Logger.Println(append([]interface{}{fmt.Sprintf("(%d)", h.jobid)}, v...)...)
	} else {
		logger.Logger.Println(v...)
	}
	if err := AppendLog(h.logfile, v...); err != nil {
		logger.Logger.Print(err)
	}
}

func (h *hylasessionlog) Logf(format string, v ...interface{}) {
	h.Log(fmt.Sprintf(format, v...))
}

func (h *hylasessionlog) CommSeq() uint64 {
	return h.commseq
}

func (h *hylasessionlog) CommID() string {
	return h.commid
}

func (h *hylasessionlog) Logfile() string {
	return h.logfile
}
