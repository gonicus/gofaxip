// Copyright 2013 Alexandre Fiori
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Event Socket client that connects to FreeSWITCH to originate a new call.
package main

import (
	"fmt"
	"log"

	"github.com/fiorix/go-eventsocket/eventsocket"
)

const dest = "sofia/internal/1000%127.0.0.1"
const dialplan = "&socket(localhost:9090 async)"

func main() {
	c, err := eventsocket.Dial("localhost:8021", "ClueCon")
	if err != nil {
		log.Fatal(err)
	}
	c.Send("events json ALL")
	c.Send(fmt.Sprintf("bgapi originate %s %s", dest, dialplan))
	for {
		ev, err := c.ReadEvent()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("\nNew event")
		ev.PrettyPrint()
		if ev.Get("Answer-State") == "hangup" {
			break
		}
	}
	c.Close()
}
