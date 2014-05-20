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
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/fiorix/go-eventsocket/eventsocket"
	"gofaxlib"
	"gofaxlib/logger"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	RECVQ_FILE_FORMAT   = "fax%08d.tif"
	RECVQ_DIR           = "recvq"
	DEFAULT_FAXRCVD_CMD = "bin/faxrcvd"
	DEFAULT_DEVICE      = "freeswitch"
)

type EventSocketServer struct {
	errorChan chan error
	killChan  chan struct{}
}

func NewEventSocketServer() *EventSocketServer {
	e := new(EventSocketServer)
	e.errorChan = make(chan error)
	e.killChan = make(chan struct{})
	return e
}

// Start listening for incoming calls
func (e *EventSocketServer) Start() {
	go func() {
		err := eventsocket.ListenAndServe(gofaxlib.Config.Gofaxd.Socket, e.handler)
		if err != nil {
			e.errorChan <- err
		}
	}()
}

// Receive fatal errors that make the server stop
func (e *EventSocketServer) Errors() <-chan error {
	return e.errorChan
}

// Abort all running connections and kill the
// corresponding FreeSWITCH channels.
// TODO: Right now we have not way implemented to wait until
// all connections have closed and signal to the caller,
// so we have to wait a few seconds after calling Kill()
func (e *EventSocketServer) Kill() {
	close(e.killChan)
}

// Handle incoming call
func (e *EventSocketServer) handler(c *eventsocket.Connection) {
	logger.Logger.Println("Incoming Event Socket connection from", c.RemoteAddr())

	connectev, err := c.Send("connect") // Returns: Ganzer Event mit alles
	if err != nil {
		c.Send("exit")
		logger.Logger.Print(err)
		return
	}

	channel_uuid := uuid.Parse(connectev.Get("Unique-Id"))
	if channel_uuid == nil {
		c.Send("exit")
		logger.Logger.Print(err)
		return
	}
	defer logger.Logger.Println(channel_uuid, "Handler ending")

	// Filter and subscribe to events
	c.Send("linger")
	c.Send(fmt.Sprintf("filter Unique-ID %v", channel_uuid))
	c.Send("event plain CHANNEL_CALLSTATE CUSTOM spandsp::rxfaxnegociateresult spandsp::rxfaxpageresult spandsp::rxfaxresult")

	// Extract Caller/Callee
	recipient := connectev.Get("Variable_sip_to_user")
	cidname := connectev.Get("Channel-Caller-Id-Name")
	cidnum := connectev.Get("Channel-Caller-Id-Number")

	logger.Logger.Printf("Incoming call to %v from %v <%v>", recipient, cidname, cidnum)

	var device *Device
	if gofaxlib.Config.Gofaxd.AllocateInboundDevices {
		// Find free device
		device, err := devmanager.FindDevice(fmt.Sprintf("Receiving facsimile"))
		if err != nil {
			logger.Logger.Println(err)
			c.Execute("respond", "404", true)
			c.Send("exit")
			return
		}
		defer device.SetReady()
	}

	var used_device string
	if device != nil {
		used_device = device.Name
	} else {
		used_device = DEFAULT_DEVICE
	}

	// Query DynamicConfig
	if dc_cmd := gofaxlib.Config.Gofaxd.DynamicConfig; dc_cmd != "" {
		logger.Logger.Println("Calling DynamicConfig script", dc_cmd)
		dc, err := gofaxlib.DynamicConfig(dc_cmd, used_device, cidnum, cidname, recipient)
		if err != nil {
			logger.Logger.Println("Error calling DynamicConfig:", err)
		} else {
			if rejectval := dc.GetFirst("RejectCall"); rejectval != "" {
				switch strings.ToLower(rejectval) {
				case "true":
					fallthrough
				case "1":
					fallthrough
				case "yes":
					logger.Logger.Println("DynamicConfig decided to reject this call")
					c.Execute("respond", "404", true)
					c.Send("exit")
					return
				}
			}
		}

	}

	sessionlog, err := gofaxlib.NewSessionLogger()
	if err != nil {
		c.Send("exit")
		logger.Logger.Print(err)
		return
	}

	logger.Logger.Println(channel_uuid, "Logging events for commid", sessionlog.CommId(), "to", sessionlog.Logfile())
	sessionlog.Log("Inbound channel UUID: ", channel_uuid)
	sessionlog.Log(fmt.Sprintf("Accepting call to %v from %v <%v> with commid %v", recipient, cidname, cidnum, sessionlog.CommId()))

	if device != nil {
		// Notify faxq
		gofaxlib.Faxq.ModemStatus(device.Name, "I"+sessionlog.CommId())
		gofaxlib.Faxq.ReceiveStatus(device.Name, "B")
		gofaxlib.Faxq.ReceiveStatus(device.Name, "S")
		defer gofaxlib.Faxq.ReceiveStatus(device.Name, "E")
	}

	// Start interacting with the caller

	if gofaxlib.Config.Gofaxd.Answerafter != 0 {
		c.Execute("ring_ready", "", true)
		c.Execute("sleep", strconv.FormatUint(gofaxlib.Config.Gofaxd.Answerafter, 10), true)
	}

	c.Execute("answer", "", true)

	if gofaxlib.Config.Gofaxd.Waittime != 0 {
		c.Execute("playback", "silence_stream://"+strconv.FormatUint(gofaxlib.Config.Gofaxd.Waittime, 10), true)
	}

	// Find filename in recvq to save received .tif
	seq, err := gofaxlib.GetSeqFor(RECVQ_DIR)
	if err != nil {
		c.Send("exit")
		sessionlog.Log(err)
		return
	}
	filename := filepath.Join(RECVQ_DIR, fmt.Sprintf(RECVQ_FILE_FORMAT, seq))
	filename_abs := filepath.Join(gofaxlib.Config.Hylafax.Spooldir, filename)

	sessionlog.Log("Rxfax to", filename_abs)

	c.Execute("set", "fax_enable_t38_request=true", true)
	c.Execute("set", "fax_enable_t38=true", true)
	c.Execute("set", fmt.Sprintf("fax_ident=%s", gofaxlib.Config.Freeswitch.Ident), true)
	c.Execute("rxfax", filename_abs, true)

	result := gofaxlib.NewFaxResult(channel_uuid, sessionlog)
	es := gofaxlib.NewEventStream(c)

	pages := result.TransferredPages

EventLoop:
	for {
		select {
		case ev := <-es.Events():
			if ev.Get("Content-Type") == "text/disconnect-notice" {
				sessionlog.Log("Received disconnect message")
				//c.Close()
				//break EventLoop
			} else {
				result.AddEvent(ev)
				if result.Hangupcause != "" {
					c.Close()
					break EventLoop
				}

				if pages != result.TransferredPages {
					pages = result.TransferredPages
					if device != nil {
						gofaxlib.Faxq.ReceiveStatus(device.Name, "P")
					}
				}
			}
		case err := <-es.Errors():
			if err.Error() == "EOF" {
				sessionlog.Log("Event socket client disconnected")
			} else {
				sessionlog.Log("Error:", err)
			}
			break EventLoop
		case _ = <-e.killChan:
			sessionlog.Log("Kill reqeust received, destroying channel")
			c.Send(fmt.Sprintf("api uuid_kill %v", channel_uuid))
			c.Close()
			return
		}
	}

	if device != nil {
		gofaxlib.Faxq.ReceiveStatus(device.Name, "D")
	}
	sessionlog.Log(fmt.Sprintf("Success: %v, Hangup Cause: %v, Result: %v", result.Success, result.Hangupcause, result.ResultText))

	xfl := gofaxlib.NewXFRecord(result)
	xfl.Modem = used_device
	xfl.Filename = filename
	xfl.Destnum = recipient
	xfl.Cidnum = cidnum
	xfl.Cidname = cidname
	if err = xfl.SaveReceptionReport(); err != nil {
		sessionlog.Log(err)
	}

	// Process received file
	rcvdcmd := gofaxlib.Config.Gofaxd.FaxRcvdCmd
	if rcvdcmd == "" {
		rcvdcmd = DEFAULT_FAXRCVD_CMD
	}
	errmsg := ""
	if !result.Success {
		errmsg = result.ResultText
	}

	cmd := exec.Command(rcvdcmd, filename, used_device, sessionlog.CommId(), errmsg, cidnum, cidname, recipient)
	sessionlog.Log("Calling", cmd.Path, cmd.Args)
	if output, err := cmd.CombinedOutput(); err != nil {
		sessionlog.Log(cmd.Path, "ended with", err)
		sessionlog.Log(output)
	} else {
		sessionlog.Log(cmd.Path, "ended successfully")
	}

	return
}
