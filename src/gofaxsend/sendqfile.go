package main

import (
	"errors"
	"fmt"
	"gofaxlib"
	"gofaxlib/logger"
	"os"
	"path/filepath"
	"strconv"
)

const (
	// Status codes from Hylafax.
	SEND_RETRY = iota
	SEND_FAILED
	SEND_DONE
	SEND_REFORMAT
)

func SendQfile(qfilename string) (int, error) {
	var err error

	// Open qfile
	qf, err := OpenQfile(qfilename)
	if err != nil {
		return SEND_FAILED, errors.New(fmt.Sprintf("Cannot open qfile %v: %v", qfilename, err))
	}

	var jobid uint64

	if jobidstr := qf.GetFirst("jobid"); jobidstr != "" {
		if jobid, err = strconv.ParseUint(jobidstr, 10, 0); err != nil {
			logger.Logger.Println("Error parsing jobid")
		}
	}

	// Create FreeSWITCH Job
	faxjob := NewFaxJob()

	faxjob.Number = qf.GetFirst("number")
	faxjob.Cidnum = qf.GetFirst("faxnumber")
	faxjob.Cidname = qf.GetFirst("sender")
	faxjob.Ident = gofaxlib.Config.Freeswitch.Ident
	faxjob.Header = gofaxlib.Config.Freeswitch.Header

	if desiredec := qf.GetFirst("desiredec"); desiredec != "" {
		if ecm_mode, err := strconv.Atoi(desiredec); err == nil {
			faxjob.UseECM = ecm_mode != 0
		}
	}

	sessionlog, err := gofaxlib.NewSessionLogger()
	if err != nil {
		return SEND_FAILED, err
	}

	qf.Set("commid", sessionlog.CommId())

	logger.Logger.Println("Logging events for commid", sessionlog.CommId(), "to", sessionlog.Logfile())
	sessionlog.Log(fmt.Sprintf("Processing HylaFAX Job %d as %v", jobid, faxjob.UUID))

	// Add TIFFs from queue file
	faxparts := qf.GetAll("fax")
	if len(faxparts) == 0 {
		return SEND_FAILED, errors.New("No fax file(s) found in qfile")
	}

	faxfile := FaxFile{}
	for _, fileentry := range faxparts {
		err := faxfile.AddItem(fileentry)
		if err != nil {
			return SEND_FAILED, err
		}
	}

	// Merge TIFFs
	faxjob.Filename = filepath.Join(os.TempDir(), "gofaxsend_"+faxjob.UUID.String()+".tif")
	defer os.Remove(faxjob.Filename)

	if err := faxfile.WriteTo(faxjob.Filename); err != nil {
		return SEND_FAILED, err
	}

	// Total attempted calls
	totdials, err := strconv.Atoi(qf.GetFirst("totdials"))
	if err != nil {
		totdials = 0
	}

	// Consecutive failed attempts to place a call
	ndials, err := strconv.Atoi(qf.GetFirst("ndials"))
	if err != nil {
		ndials = 0
	}

	// Total answered calls
	tottries, err := strconv.Atoi(qf.GetFirst("tottries"))
	if err != nil {
		tottries = 0
	}

	// Send job
	qf.Set("status", "Dialing")
	totdials++
	qf.Set("totdials", strconv.Itoa(totdials))
	if err = qf.Write(); err != nil {
		sessionlog.Log("%v Error updating qfile %v", faxjob.UUID, qf.filename)
	}

	t := Transmit(*faxjob, sessionlog)
	var result *gofaxlib.FaxResult

	// Wait for events
	returned := SEND_RETRY
	done := false
	var faxerr FaxError

	for {
		select {
		case page := <-t.PageSent():
			// Update qfile
			qf.Set("npages", strconv.Itoa(int(page.Page)))
			qf.Set("dataformat", page.EncodingName)

		case result = <-t.Result():
			qf.Set("signalrate", strconv.Itoa(int(result.TransferRate)))
			qf.Set("csi", result.RemoteID)

			if result.Hangupcause != "" {
				// Fax Finished
				done = true
				qf.Set("status", result.ResultText)
				if result.Success {
					qf.Set("returned", strconv.Itoa(SEND_DONE))
					returned = SEND_DONE
					sessionlog.Log(fmt.Sprintf("Success: %v, Hangup Cause: %v, Result: %v", result.Success, result.Hangupcause, result.ResultText))
				}
			} else {
				// Negotiation finished
				negstatus := fmt.Sprint("Sending ", result.TransferRate)
				if result.Ecm {
					negstatus = negstatus + "/ECM"
				}
				qf.Set("status", negstatus)
				tottries++
				qf.Set("totdials", strconv.Itoa(tottries))
				ndials = 0
				qf.Set("ndials", strconv.Itoa(ndials))
			}

		case faxerr = <-t.Errors():
			done = true
			ndials++
			qf.Set("ndials", strconv.Itoa(ndials))
			qf.Set("status", faxerr.Error())
			if faxerr.Retry() {
				returned = SEND_RETRY
			} else {
				returned = SEND_FAILED
			}
		}

		if err = qf.Write(); err != nil {
			sessionlog.Log("%v Error updating qfile %v", faxjob.UUID, qf.filename)
		}

		if done {
			break
		}
	}

	xfl := gofaxlib.NewXFRecord(result)
	xfl.Modem = *device_id
	xfl.Jobid = uint(jobid)
	xfl.Jobtag = qf.GetFirst("jobtag")
	xfl.Sender = qf.GetFirst("mailaddr")
	xfl.Destnum = faxjob.Number
	xfl.Owner = qf.GetFirst("owner")
	if err = xfl.SaveTransmissionReport(); err != nil {
		sessionlog.Log(err)
	}

	return returned, faxerr
}
