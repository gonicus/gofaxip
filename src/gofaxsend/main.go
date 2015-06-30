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
	"flag"
	"fmt"
	"gofaxlib"
	"gofaxlib/logger"
	"log"
	"os"
	"path/filepath"
)

const (
	defaultConfigfile = "/etc/gofax.conf"
	productName       = "GOfax.IP"
	fifoPrefix        = "FIFO."
)

var (
	configFile  = flag.String("c", defaultConfigfile, "GOfax configuration file")
	deviceID    = flag.String("m", "", "Virtual modem device ID")
	showVersion = flag.Bool("version", false, "Show version information")

	usage = fmt.Sprintf("Usage: %s -version | [-c configfile] -m deviceID qfile [qfile [qfile [...]]]", os.Args[0])

	// Version can be set at build time using:
	//    -ldflags "-X main.version 0.42"
	version string
)

func init() {
	if version == "" {
		version = "development version"
	}
	version = fmt.Sprintf("%v %v", productName, version)

	flag.Usage = func() {
		log.Printf("%s\n%s\n", version, usage)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(1)
	}

	if *deviceID == "" || !(flag.NArg() > 0) {
		logger.Logger.Print(usage)
		log.Fatal(usage)
	}

	gofaxlib.LoadConfig(*configFile)

	var err error
	returned := 1 // Exit code

	devicefifo := filepath.Join(gofaxlib.Config.Hylafax.Spooldir, fifoPrefix+*deviceID)
	gofaxlib.SendFIFO(devicefifo, "SB")

	for _, qfilename := range flag.Args() {
		returned, err = SendQfile(qfilename)

		if err != nil {
			logger.Logger.Printf("Error processing qfile %v: %v", qfilename, err)
			break
		}
	}

	gofaxlib.SendFIFO(devicefifo, "SR")

	logger.Logger.Print("Exiting with status ", returned)
	os.Exit(returned)
}
