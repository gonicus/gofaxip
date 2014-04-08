package main

import (
	"flag"
	"fmt"
	"gofaxlib"
	//"gofaxlib/device"
	"gofaxlib/logger"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	DEFAULT_CONFIGFILE = "/etc/gofax.conf"
	PRODUCT_NAME       = "GOfax.IP"
)

var (
	config_file  = flag.String("c", DEFAULT_CONFIGFILE, "GOfax configuration file")
	show_version = flag.Bool("version", false, "Show version information")

	usage = fmt.Sprintf("Usage: %s -version | [-c configfile]", os.Args[0])

	// Version can be set at build time using:
	//    -ldflags "-X main.version 0.42"
	version string
)

func init() {
	if version == "" {
		version = "development version"
	}

	flag.Usage = func() {
		log.Printf("%s %s\n%s\n", PRODUCT_NAME, version, usage)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if *show_version {
		fmt.Println(version)
		os.Exit(1)
	}

	logger.Logger.Printf("%v gofaxd %v starting", PRODUCT_NAME, version)
	gofaxlib.LoadConfig(*config_file)

	if err := os.Chdir(gofaxlib.Config.Hylafax.Spooldir); err != nil {
		logger.Logger.Print(err)
		log.Fatal(err)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGINT)

	server := NewEventSocketServer()
	server.Start()

	// Block until something happens
	select {
	case err := <-server.Errors():
		logger.Logger.Print(err)
		log.Fatal(err)
	case sig := <-sigchan:
		logger.Logger.Print("Received ", sig, ", killing all channels")
		server.Kill()
		time.Sleep(3 * time.Second)
		logger.Logger.Print("Terminating")
		os.Exit(1)
	}

	// _, err := device.NewDevice("freeswitch1")
	// if err != nil {
	// 	log.Fatal(err)
	// }

}
