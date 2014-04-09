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
		Cidname  string
	}
	Hylafax struct {
		Spooldir string
		Modems   uint64
	}
	Gofaxd struct {
		Socket        string
		Answerafter   uint64
		Waittime      uint64
		FaxRcvdCmd    string
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
