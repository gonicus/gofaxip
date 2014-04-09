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
	LOG_DIR             = "log"
	COMMID_FORMAT       = "%08d"
	LOG_FILE_FORMAT     = "c%s"
	RECVQ_FILE_FORMAT   = "fax%08d.tif"
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

	// Filter and subscribe to events
	c.Send("linger")
	c.Send(fmt.Sprintf("filter Unique-ID %v", channel_uuid))
	c.Send("event plain CHANNEL_CALLSTATE CUSTOM spandsp::rxfaxnegociateresult spandsp::rxfaxpageresult spandsp::rxfaxresult")

	// Fetch commid and log file name
	commseq, err := gofaxlib.GetSeqFor(LOG_DIR)
	if err != nil {
		c.Send("exit")
		logger.Logger.Print(err)
		return
	}
	commid := fmt.Sprintf(COMMID_FORMAT, commseq)
	logfile := filepath.Join(LOG_DIR, fmt.Sprintf(LOG_FILE_FORMAT, commid))

	// Create a logging function that will log both to syslog and the session
	// log file for this commid
	logfunc := func(v ...interface{}) {
		uuid_values := append([]interface{}{channel_uuid}, v...)
		logger.Logger.Println(uuid_values...)
		if logerr := gofaxlib.AppendLog(logfile, v...); logerr != nil {
			logger.Logger.Print(logerr)
		}
	}

	// Extract Caller/Callee
	recipient := connectev.Get("Variable_sip_to_user")
	cidname := connectev.Get("Channel-Caller-Id-Name")
	cidnum := connectev.Get("Channel-Caller-Id-Number")
	logfunc(fmt.Sprintf("Incoming call to %v from %v <%v>", recipient, cidname, cidnum))

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
		logfunc(err)
		return
	}
	filename := filepath.Join(RECVQ_DIR, fmt.Sprintf(RECVQ_FILE_FORMAT, seq))
	filename_abs := filepath.Join(gofaxlib.Config.Hylafax.Spooldir, filename)

	logfunc("Rxfax to", filename_abs)

	c.Execute("set", "fax_enable_t38_request=true", true)
	c.Execute("set", "fax_enable_t38=true", true)
	c.Execute("set", fmt.Sprintf("fax_ident=%s", gofaxlib.Config.Freeswitch.Ident), true)
	c.Execute("rxfax", filename_abs, true)

	result := gofaxlib.NewFaxResult(channel_uuid, logfunc)
	es := gofaxlib.NewEventStream(c)

EventLoop:
	for {
		select {
		case ev := <-es.Events():
			if ev.Get("Content-Type") == "text/disconnect-notice" {
				logfunc("Received disconnect message")
				c.Close()
				break EventLoop
			}
			result.AddEvent(ev)
		case err := <-es.Errors():
			if err.Error() == "EOF" {
				logfunc("Event socket client disconnected")
			} else {
				logfunc("Error:", err)
			}
			break EventLoop
		case _ = <-e.killChan:
			logfunc("Kill reqeust received, destroying channel")
			c.Send(fmt.Sprintf("api uuid_kill %v", channel_uuid))
			c.Close()
			return
		}
	}

	logfunc(fmt.Sprintf("Success: %v, Hangup Cause: %v, Result: %v", result.Success, result.Hangupcause, result.ResultText))

	// Process received file
	rcvdcmd := gofaxlib.Config.Gofaxd.FaxRcvdCmd
	if rcvdcmd == "" {
		rcvdcmd = DEFAULT_FAXRCVD_CMD
	}
	errmsg := ""
	if !result.Success {
		errmsg = result.ResultText
	}

	cmd := exec.Command(rcvdcmd, filename, "freeswitch1", commid, errmsg, cidnum, cidname, recipient)
	logfunc("Calling", cmd.Path, cmd.Args)
	if output, err := cmd.CombinedOutput(); err != nil {
		logfunc(rcvdcmd, "returned error:", err)
		logfunc(output)
	}

	logger.Logger.Println(channel_uuid, "Handler ending")
	return
}
