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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/gonicus/gofaxip/gofaxlib"
	"github.com/gonicus/gofaxip/gofaxlib/logger"

	"github.com/fiorix/go-eventsocket/eventsocket"
	"github.com/google/uuid"
)

const (
	recvqFileFormat   = "fax%08d.tif"
	recvqDir          = "recvq"
	defaultFaxrcvdCmd = "bin/faxrcvd"
	defaultDevice     = "freeswitch"
)

// EventSocketServer is a server for handling outgoing event socket connections from FreeSWITCH
type EventSocketServer struct {
	errorChan chan error
	killChan  chan struct{}
}

// NewEventSocketServer initializes a EventSocketServer
func NewEventSocketServer() *EventSocketServer {
	e := new(EventSocketServer)
	e.errorChan = make(chan error)
	e.killChan = make(chan struct{})
	return e
}

func getNumberFromSIPURI(uri string) (string, error) {
	re := regexp.MustCompile(`sip:(\d+)@`)

	matches := re.FindStringSubmatch(uri)

	if len(matches) < 2 {
		return "", fmt.Errorf("Number could not be extracted from SIP URI")
	}

	number := matches[1]
	return number, nil
}

// Start starts a goroutine to listen for ESL connections and handle incoming calls
func (e *EventSocketServer) Start() {
	go func() {
		err := eventsocket.ListenAndServe(gofaxlib.Config.Gofaxd.Socket, e.handler)
		if err != nil {
			e.errorChan <- err
		}
	}()
}

// Errors returns a channel of fatal errors that make the server stop
func (e *EventSocketServer) Errors() <-chan error {
	return e.errorChan
}

// Kill aborts all running connections and kills the
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

	connectev, err := c.Send("connect") // Returns a whole event
	if err != nil {
		c.Send("exit")
		logger.Logger.Print(err)
		return
	}

	channelUUID, err := uuid.Parse(connectev.Get("Unique-Id"))
	if err != nil {
		c.Send("exit")
		logger.Logger.Print(err)
		return
	}
	defer logger.Logger.Println(channelUUID, "Handler ending")

	// Filter and subscribe to events
	c.Send("linger")
	c.Send(fmt.Sprintf("filter Unique-ID %v", channelUUID))
	c.Send("event plain CHANNEL_CALLSTATE CUSTOM spandsp::rxfaxnegociateresult spandsp::rxfaxpageresult spandsp::rxfaxresult")

	// Extract Caller/Callee
	var recipient string
	if gofaxlib.Config.Gofaxd.RecipientFromDiversionHeader {
		recipient, err = getNumberFromSIPURI(connectev.Get("Variable_sip_h_diversion"))
		if err != nil {
			logger.Logger.Println(err)
			c.Execute("respond", "404", true)
			c.Send("exit")
			return
		}
	} else {
		recipient = connectev.Get("Variable_sip_to_user")
	}

	gateway := connectev.Get("Variable_sip_gateway")
	cidname := connectev.Get("Channel-Caller-Id-Name")
	cidnum := connectev.Get("Channel-Caller-Id-Number")

	logger.Logger.Printf("Incoming call to %v from %v <%v> via gateway %v", recipient, cidname, cidnum, gateway)

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

	var usedDevice string
	if device != nil {
		usedDevice = device.Name
	} else {
		usedDevice = defaultDevice
	}

	csi := gofaxlib.Config.Freeswitch.Ident

	// Query DynamicConfig
	if dcCmd := gofaxlib.Config.Gofaxd.DynamicConfig; dcCmd != "" {
		logger.Logger.Println("Calling DynamicConfig script", dcCmd)
		dc, err := gofaxlib.DynamicConfig(dcCmd, usedDevice, cidnum, cidname, recipient, gateway)
		if err != nil {
			logger.Logger.Println("Error calling DynamicConfig:", err)
		} else {
			// Check if call should be rejected
			if gofaxlib.DynamicConfigBool(dc.GetString("RejectCall")) {
				logger.Logger.Println("DynamicConfig decided to reject this call")
				c.Execute("respond", "404", true)
				c.Send("exit")
				return
			}

			// Check if a custom identifier should be set
			if dynamicCsi := dc.GetString("LocalIdentifier"); dynamicCsi != "" {
				csi = dynamicCsi
			}

		}
	}

	sessionlog, err := gofaxlib.NewSessionLogger(0)
	if err != nil {
		c.Send("exit")
		logger.Logger.Print(err)
		return
	}

	logger.Logger.Println(channelUUID, "Logging events for commid", sessionlog.CommID(), "to", sessionlog.Logfile())
	sessionlog.Log("Inbound channel UUID: ", channelUUID)

	// Check if T.38 should be enabled
	requestT38 := gofaxlib.Config.Gofaxd.RequestT38
	enableT38 := gofaxlib.Config.Gofaxd.EnableT38

	fallback, err := gofaxlib.GetSoftmodemFallback(nil, cidnum)
	if err != nil {
		sessionlog.Log(err)
	}
	if fallback {
		sessionlog.Logf("Softmodem fallback active for caller %s, disabling T.38", cidnum)
		enableT38 = false
		requestT38 = false
	}
	sessionlog.Logf("Accepting call to %v from %v <%v> via gateway %v with commid %v", recipient, cidname, cidnum, gateway, sessionlog.CommID())

	if device != nil {
		// Notify faxq
		gofaxlib.Faxq.ModemStatus(device.Name, "I"+sessionlog.CommID())
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
	seq, err := gofaxlib.GetSeqFor(recvqDir)
	if err != nil {
		c.Send("exit")
		sessionlog.Log(err)
		return
	}
	filename := filepath.Join(recvqDir, fmt.Sprintf(recvqFileFormat, seq))
	filenameAbs := filepath.Join(gofaxlib.Config.Hylafax.Spooldir, filename)

	sessionlog.Log("Rxfax to", filenameAbs)

	c.Execute("set", fmt.Sprintf("fax_enable_t38=%s", strconv.FormatBool(enableT38)), true)
	c.Execute("set", fmt.Sprintf("fax_enable_t38_request=%s", strconv.FormatBool(requestT38)), true)
	c.Execute("set", fmt.Sprintf("fax_ident=%s", csi), true)
	c.Execute("rxfax", filenameAbs, true)
	c.Execute("hangup", "", true)

	result := gofaxlib.NewFaxResult(channelUUID, sessionlog)
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
			c.Send(fmt.Sprintf("api uuid_kill %v", channelUUID))
			c.Close()
			return
		}
	}

	if device != nil {
		gofaxlib.Faxq.ReceiveStatus(device.Name, "D")
	}
	sessionlog.Logf("Success: %v, Hangup Cause: %v, Result: %v", result.Success, result.Hangupcause, result.ResultText)

	xfl := &gofaxlib.XFRecord{}
	xfl.Commid = sessionlog.CommID()
	xfl.SetResult(result)
	xfl.Modem = usedDevice
	xfl.Filename = filename
	xfl.Destnum = recipient
	xfl.Cidnum = cidnum
	xfl.Cidname = cidname
	if err = xfl.SaveReceptionReport(); err != nil {
		sessionlog.Log(err)
	}

	// If reception failed:
	// Check if softmodem fallback should be enabled on the next call
	if gofaxlib.Config.Freeswitch.SoftmodemFallback && !result.Success {
		var activateFallback bool

		if result.NegotiateCount > 1 {
			// Activate fallback if negotiation was repeated
			sessionlog.Logf("Fax failed with %d negotiations, enabling softmodem fallback for calls from/to %s.", result.NegotiateCount, cidnum)
			activateFallback = true
		} else {
			var badrows uint
			for _, p := range result.PageResults {
				badrows += p.BadRows
			}
			if badrows > 0 {
				// Activate fallback if any bad rows were present
				sessionlog.Logf("Fax failed with %d bad rows in %d pages, enabling softmodem fallback for calls from/to %s.", badrows, result.TransferredPages, cidnum)
				activateFallback = true
			}
		}

		if activateFallback {
			err = gofaxlib.SetSoftmodemFallback(nil, cidnum, true)
			if err != nil {
				sessionlog.Log(err)
			}
		}

	}

	// Process received file
	rcvdcmd := gofaxlib.Config.Gofaxd.FaxRcvdCmd
	if rcvdcmd == "" {
		rcvdcmd = defaultFaxrcvdCmd
	}
	errmsg := ""
	if !result.Success {
		errmsg = result.ResultText
	}

	cmd := exec.Command(rcvdcmd, filename, usedDevice, sessionlog.CommID(), errmsg, cidnum, cidname, recipient, gateway)
	extraEnv := []string{
		fmt.Sprintf("HANGUPCAUSE=%s", result.Hangupcause),
		fmt.Sprintf("TRANSFER_RATE=%d", result.TransferRate),
	}
	cmd.Env = append(os.Environ(), extraEnv...)
	sessionlog.Log("Calling", cmd.Path, cmd.Args)
	if output, err := cmd.CombinedOutput(); err != nil {
		sessionlog.Log(cmd.Path, "ended with", err)
		sessionlog.Log(output)
	} else {
		sessionlog.Log(cmd.Path, "ended successfully")
	}

	return
}
