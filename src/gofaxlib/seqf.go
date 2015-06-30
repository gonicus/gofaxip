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
	"os"
	"path/filepath"
	"syscall"
)

const (
	seqFileName = "seqf"
)

// GetSeqFor increments and returns the sequence number for given HylaFAX spool area
func GetSeqFor(subdir string) (seq uint64, err error) {
	seqfname := filepath.Join(Config.Hylafax.Spooldir, subdir, seqFileName)

	seqf, err := os.OpenFile(seqfname, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	defer seqf.Close()

	if err = syscall.Flock(int(seqf.Fd()), syscall.LOCK_EX); err != nil {
		return
	}

	if _, err = fmt.Fscan(seqf, &seq); err != nil && err.Error() != "EOF" {
		return
	}

	seq++

	if err = seqf.Truncate(0); err != nil {
		return
	}
	if _, err = seqf.Seek(0, 0); err != nil {
		return
	}

	if _, err = fmt.Fprintf(seqf, "%d\n", seq); err != nil {
		return
	}

	return
}
