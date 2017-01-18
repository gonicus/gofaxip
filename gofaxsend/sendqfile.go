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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gonicus/gofaxip/gofaxlib"
	"github.com/gonicus/gofaxip/gofaxlib/logger"
)

const (
	// Status codes from Hylafax.
	sendRetry = iota
	sendFailed
	sendDone
	sendReformat
)

// SendQfile immediately tries to send the given qfile using FreeSWITCH
func SendQfile(qfilename string) (int, error) {
	var err error

	// Open qfile
	qf, err := OpenQfile(qfilename)
	if err != nil {
		return sendFailed, fmt.Errorf("Cannot open qfile %v: %v", qfilename, err)
	}
	defer qf.Close()

	var jobid uint

	jobidstr := qf.GetString("jobid")
	if jobidstr != "" {
		if i, err := strconv.Atoi(jobidstr); err == nil {
			jobid = uint(i)
		} else {
			logger.Logger.Println("Error parsing jobid")
		}
	}

	// Create FreeSWITCH Job
	faxjob, err := NewFaxJob()
	if err != nil {
		return sendFailed, fmt.Errorf("Cannot create fax job: %s", err)
	}

	faxjob.Number = fmt.Sprint(gofaxlib.Config.Gofaxsend.CallPrefix, qf.GetString("number"))
	faxjob.Cidnum = gofaxlib.Config.Gofaxsend.FaxNumber //qf.GetString("faxnumber")
	faxjob.Cidname = qf.GetString("sender")
	faxjob.Ident = gofaxlib.Config.Freeswitch.Ident
	faxjob.Header = gofaxlib.Config.Freeswitch.Header
	faxjob.Gateways = gofaxlib.Config.Freeswitch.Gateway

	if ecmMode, err := qf.GetInt("desiredec"); err == nil {
		faxjob.UseECM = ecmMode != 0
	}

	if brMode, err := qf.GetInt("desiredbr"); err == nil {
		if brMode < 5 { // < 14400bps
			faxjob.DisableV17 = true
		}
	}

	// Query DynamicConfig
	if dcCmd := gofaxlib.Config.Gofaxsend.DynamicConfig; dcCmd != "" {
		logger.Logger.Println("Calling DynamicConfig script", dcCmd)
		dc, err := gofaxlib.DynamicConfig(dcCmd, *deviceID, qf.GetString("owner"), qf.GetString("number"), jobidstr)
		if err != nil {
			errmsg := fmt.Sprintln("Error calling DynamicConfig:", err)
			logger.Logger.Println(errmsg)
			qf.Set("status", errmsg)
			if err = qf.Write(); err != nil {
				logger.Logger.Println("Error updating qfile:", err)
			}
			return sendFailed, errors.New(errmsg)

		}

		// Check if call should be rejected
		if gofaxlib.DynamicConfigBool(dc.GetString("RejectCall")) {
			errmsg := "Transmission rejected by DynamicConfig"
			logger.Logger.Println(errmsg)
			qf.Set("status", errmsg)
			if err = qf.Write(); err != nil {
				logger.Logger.Println("Error updating qfile:", err)
			}
			return sendFailed, errors.New(errmsg)
		}

		// Check if a custom identifier should be set
		if dynamicTsi := dc.GetString("LocalIdentifier"); dynamicTsi != "" {
			faxjob.Ident = dynamicTsi
		}

		if tagline := dc.GetString("TagLine"); tagline != "" {
			faxjob.Header = tagline
		}

		if faxnumber := dc.GetString("FAXNumber"); faxnumber != "" {
			faxjob.Cidnum = faxnumber
		}

		if gatewayString := dc.GetString("Gateway"); gatewayString != "" {
			faxjob.Gateways = strings.Split(gatewayString, ",")
		}

	}

	// Start session
	sessionlog, err := gofaxlib.NewSessionLogger()
	if err != nil {
		return sendFailed, err
	}

	qf.Set("commid", sessionlog.CommID())

	logger.Logger.Println("Logging events for commid", sessionlog.CommID(), "to", sessionlog.Logfile())
	sessionlog.Logf("Processing HylaFAX Job %d as %v", jobid, faxjob.UUID)

	// Add TIFFs from queue file
	faxparts := qf.GetAll("fax")
	if len(faxparts) == 0 {
		return sendFailed, errors.New("No fax file(s) found in qfile")
	}

	faxfile := FaxFile{}
	for _, fileentry := range faxparts {
		err := faxfile.AddItem(fileentry)
		if err != nil {
			return sendFailed, err
		}
	}

	// Merge TIFFs
	faxjob.Filename = filepath.Join(os.TempDir(), "gofaxsend_"+faxjob.UUID.String()+".tif")
	defer os.Remove(faxjob.Filename)

	if err := faxfile.WriteTo(faxjob.Filename); err != nil {
		return sendFailed, err
	}

	// Total attempted calls
	totdials, _ := qf.GetInt("totdials")

	// Consecutive failed attempts to place a call
	ndials, _ := qf.GetInt("ndials")

	// Total answered calls
	tottries, _ := qf.GetInt("tottries")

	// Send job
	qf.Set("status", "Dialing")
	totdials++
	qf.Set("totdials", strconv.Itoa(totdials))
	if err = qf.Write(); err != nil {
		sessionlog.Log("Error updating qfile:", err)
	}

	t := transmit(*faxjob, sessionlog)
	var result *gofaxlib.FaxResult

	// Wait for events
	returned := sendRetry
	done := false
	var faxerr FaxError

	for !done {
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
					qf.Set("returned", strconv.Itoa(sendDone))
					returned = sendDone
					sessionlog.Logf("Success: %v, Hangup Cause: %v, Result: %v", result.Success, result.Hangupcause, result.ResultText)
				}
			} else {
				// Negotiation finished
				negstatus := fmt.Sprint("Sending ", result.TransferRate)
				if result.Ecm {
					negstatus = negstatus + "/ECM"
				}
				qf.Set("status", negstatus)
				tottries++
				qf.Set("tottries", strconv.Itoa(tottries))
				ndials = 0
				qf.Set("ndials", strconv.Itoa(ndials))
			}

		case faxerr = <-t.Errors():
			done = true
			ndials++
			qf.Set("ndials", strconv.Itoa(ndials))
			qf.Set("status", faxerr.Error())
			if faxerr.Retry() {
				returned = sendRetry
			} else {
				returned = sendFailed
			}
		}

		if err = qf.Write(); err != nil {
			sessionlog.Log("Error updating qfile:", err)
		}
	}

	if result != nil {
		xfl := gofaxlib.NewXFRecord(result)
		xfl.Modem = *deviceID
		xfl.Jobid = uint(jobid)
		xfl.Jobtag = qf.GetString("jobtag")
		xfl.Sender = qf.GetString("mailaddr")
		xfl.Destnum = qf.GetString("number")
		xfl.Owner = qf.GetString("owner")
		if err = xfl.SaveTransmissionReport(); err != nil {
			sessionlog.Log(err)
		}
	}

	return returned, faxerr
}
