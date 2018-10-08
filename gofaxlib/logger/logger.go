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

package logger

import (
	"log"
	"log/syslog"
	"os"
)

const (
	LOG_PRIORITY = syslog.LOG_DAEMON | syslog.LOG_INFO
	LOG_FLAGS    = log.Lshortfile
)

var (
	Logger *log.Logger
)

func init() {
	var err error
	log.SetFlags(LOG_FLAGS)

	if os.Getenv("CI") != "" {
		// Running in Circle CI
		Logger = log.New(os.Stderr, "", LOG_FLAGS)
		return
	}

	Logger, err = syslog.NewLogger(LOG_PRIORITY, LOG_FLAGS)
	if err != nil {
		log.Fatal(err)
	}

}
