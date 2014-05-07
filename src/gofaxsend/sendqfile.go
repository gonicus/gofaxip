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

type HylafaxMetadata struct {
	Commid string
	Jobid  uint
	Jobtag string
}

func SendQfile(devicename string, qfilename string) (int, error) {
	var err error

	// Open qfile
	qf, err := OpenQfile(qfilename)
	if err != nil {
		return SEND_FAILED, errors.New(fmt.Sprintf("Cannot open qfile %v: %v", qfilename, err))
	}

	// Get Metadata from qfile
	meta := &HylafaxMetadata{
		Jobtag: qf.GetFirst("jobtag"),
		Commid: qf.GetFirst("commid"),
	}

	if jobid := qf.GetFirst("jobid"); jobid != "" {
		if jobid, err := strconv.ParseUint(jobid, 10, 0); err == nil {
			meta.Jobid = uint(jobid)
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

	logger.Logger.Printf("Processing HylaFAX Job %d as %v", meta.Jobid, faxjob.UUID)

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
		logger.Logger.Printf("%v Error updating qfile %v", faxjob.UUID, qf.filename)
	}

	t := Transmit(*faxjob)
	var result gofaxlib.FaxResult

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
					logger.Logger.Printf("%v Success: %v, Hangup Cause: %v, Result: %v", faxjob.UUID, result.Success, result.Hangupcause, result.ResultText)
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
			logger.Logger.Printf("%v Error updating qfile %v", faxjob.UUID, qf.filename)
		}

		if done {
			break
		}
	}

	return returned, faxerr
}
