package main

import (
	"flag"
	"fmt"
	"gofaxlib"
	"gofaxlib/logger"
	"log"
	"os"
)

const (
	DEFAULT_CONFIGFILE = "/etc/gofax.conf"
	PRODUCT_NAME       = "GOfax.IP"
)

var (
	config_file  = flag.String("c", DEFAULT_CONFIGFILE, "GOfax configuration file")
	device_id    = flag.String("m", "", "Virtual modem device ID")
	show_version = flag.Bool("version", false, "Show version information")

	usage = fmt.Sprintf("Usage: %s -version | [-c configfile] -m deviceID qfile [qfile [qfile [...]]]", os.Args[0])

	// Version can be set at build time using:
	//    -ldflags "-X main.version 0.42"
	version string
)

func init() {
	if version == "" {
		version = "development version"
	}
	version = fmt.Sprintf("%v %v", PRODUCT_NAME, version)

	flag.Usage = func() {
		log.Printf("%s\n%s\n", version, usage)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if *show_version {
		fmt.Println(version)
		os.Exit(1)
	}

	if *device_id == "" || !(flag.NArg() > 0) {
		logger.Logger.Print(usage)
		log.Fatal(usage)
	}

	gofaxlib.LoadConfig(*config_file)

	var err error
	returned := 1 // Exit code
	for _, qfilename := range flag.Args() {
		returned, err = SendQfile(*device_id, qfilename)

		if err != nil {
			logger.Logger.Printf("Error processing qfile %v: %v", qfilename, err)
		}
	}

	logger.Logger.Print("Exiting with status ", returned)
	os.Exit(returned)
}
