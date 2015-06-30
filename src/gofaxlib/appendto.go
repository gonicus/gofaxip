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
	"log"
	"os"
	"syscall"
)

const (
	appendLogFlags = log.Ldate | log.Ltime | log.Lmicroseconds
)

// AppendTo appends a line to file, adding line break
func AppendTo(filename string, line string) (err error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return
	}

	if _, err = f.WriteString(line + "\n"); err != nil {
		return
	}
	return
}

// AppendLog appends a log message to file, adding line break and timestamp
func AppendLog(filename string, v ...interface{}) (err error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return
	}

	l := log.New(f, "", appendLogFlags)
	l.Println(v...)

	return
}
