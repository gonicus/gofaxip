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
	"code.google.com/p/gcfg"
	"gofaxlib/logger"
	"log"
)

var (
	Config config
)

type config struct {
	Freeswitch struct {
		Socket   string
		Password string
		Gateway  []string
		Ident    string
		Header   string
		Verbose  bool
	}
	Hylafax struct {
		Spooldir   string
		Modems     uint
		Xferfaxlog string
	}
	Gofaxd struct {
		Socket                 string
		Answerafter            uint64
		Waittime               uint64
		FaxRcvdCmd             string
		DynamicConfig          string
		AllocateInboundDevices bool
	}
	Gofaxsend struct {
		FaxNumber     string
		DynamicConfig string
	}
}

func LoadConfig(filename string) {
	err := gcfg.ReadFileInto(&Config, filename)
	if err != nil {
		logger.Logger.Print("Config: ", err)
		log.Fatal("Config: ", err)
	}
}
