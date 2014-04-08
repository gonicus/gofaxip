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
)

const (
	RECVQ_FILE_FORMAT   = "fax%09d.tif"
	RECVQ_DIR           = "recvq"
	DEFAULT_FAXRCVD_CMD = "bin/faxrcvd"
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
	logger.Logger.Print("Incoming Event Socket connection from ", c.RemoteAddr())

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

	recipient := connectev.Get("Variable_sip_to_user")
	cidname := connectev.Get("Channel-Caller-Id-Name")
	cidnum := connectev.Get("Channel-Caller-Id-Number")

	logger.Logger.Printf("%v Incoming call to %v from %v <%v>", channel_uuid, recipient, cidname, cidnum)

	c.Send("linger")
	c.Send(fmt.Sprintf("filter Unique-ID %v", channel_uuid))
	c.Send("event plain CHANNEL_CALLSTATE CUSTOM spandsp::rxfaxnegociateresult spandsp::rxfaxpageresult spandsp::rxfaxresult")

	if gofaxlib.Config.Gofaxd.Answerafter != 0 {
		c.Execute("ring_ready", "", true)
		c.Execute("sleep", strconv.FormatUint(gofaxlib.Config.Gofaxd.Answerafter, 10), true)
	}

	c.Execute("answer", "", true)

	if gofaxlib.Config.Gofaxd.Waittime != 0 {
		c.Execute("playback", "silence_stream://"+strconv.FormatUint(gofaxlib.Config.Gofaxd.Waittime, 10), true)
	}

	c.Execute("set", "fax_enable_t38_request=true", true)
	c.Execute("set", "fax_enable_t38=true", true)

	seq, err := gofaxlib.GetSeqFor(RECVQ_DIR)
	if err != nil {
		c.Send("exit")
		logger.Logger.Print(err)
		return
	}
	filename := filepath.Join(RECVQ_DIR, fmt.Sprintf(RECVQ_FILE_FORMAT, seq))
	filename_abs := filepath.Join(gofaxlib.Config.Hylafax.Spooldir, filename)

	logger.Logger.Printf("%v Rxfax to %v", channel_uuid, filename_abs)
	c.Execute("rxfax", filename_abs, true)

	result := gofaxlib.NewFaxResult(channel_uuid)
	es := gofaxlib.NewEventStream(c)

EventLoop:
	for {
		select {
		case ev := <-es.Events():
			if ev.Get("Content-Type") == "text/disconnect-notice" {
				logger.Logger.Printf("%v Received disconnect message", channel_uuid)
				c.Close()
				break EventLoop
			}
			result.AddEvent(ev)
		case err := <-es.Errors():
			if err.Error() == "EOF" {
				logger.Logger.Printf("%v Event socket client disconnected", channel_uuid)
			} else {
				logger.Logger.Print(channel_uuid, " Error: ", err)
			}
			break EventLoop
		case _ = <-e.killChan:
			logger.Logger.Printf("%v Kill reqeust received, destroying channel", channel_uuid)
			c.Send(fmt.Sprintf("api uuid_kill %v", channel_uuid))
			c.Close()
			return
		}
	}

	logger.Logger.Printf("%v Success: %v, Hangup Cause: %v, Result: %v", channel_uuid, result.Success, result.Hangupcause, result.ResultText)

	// Process received file
	rcvdcmd := gofaxlib.Config.Gofaxd.FaxRcvdCmd
	if rcvdcmd == "" {
		rcvdcmd = DEFAULT_FAXRCVD_CMD
	}
	errmsg := ""
	if !result.Success {
		errmsg = result.ResultText
	}

	logger.Logger.Printf("%v running %v", channel_uuid, rcvdcmd)
	cmd := exec.Command(rcvdcmd, filename, "freeswitch1", fmt.Sprintf("%09d", seq), errmsg, cidnum, cidname, recipient)
	_, err = cmd.CombinedOutput()
	if err != nil {
		logger.Logger.Printf("%s %s returned error: %v", channel_uuid, rcvdcmd, err)
	}

	logger.Logger.Printf("%s Handler ending", channel_uuid)
	return
}
