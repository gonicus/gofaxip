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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gonicus/gofaxip/gofaxlib"
	"github.com/gonicus/gofaxip/gofaxlib/logger"
)

const (
	// Return codes for Hylafax.
	sendRetry = iota
	sendFailed
	sendDone
	sendReformat
)

// SendQfile immediately tries to send the given qfile using FreeSWITCH
func SendQfile(qfilename string) (returned int, err error) {
	returned = sendFailed

	// Open qfile
	qf, err := OpenQfile(qfilename)
	if err != nil {
		err = fmt.Errorf("Cannot open qfile %v: %v", qfilename, err)
		return
	}
	defer qf.Close()

	var jobid uint
	if jobidstr := qf.GetString("jobid"); jobidstr != "" {
		if i, err := strconv.Atoi(jobidstr); err == nil {
			jobid = uint(i)
		}
	}

	if jobid == 0 {
		err = fmt.Errorf("Error parsing jobid")
		return
	}

	// Create FaxJob structure
	faxjob := NewFaxJob()
	faxjob.Number = fmt.Sprint(gofaxlib.Config.Gofaxsend.CallPrefix, qf.GetString("external"))
	faxjob.Cidnum = gofaxlib.Config.Gofaxsend.FaxNumber //qf.GetString("faxnumber")
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

	// Add TIFFs from queue file
	faxparts := qf.GetAll("fax")
	if len(faxparts) == 0 {
		err = fmt.Errorf("No fax file(s) found in qfile")
		return
	}
	faxfile := FaxFile{}
	for _, fileentry := range faxparts {
		err = faxfile.AddItem(fileentry)
		if err != nil {
			return
		}
	}

	// Merge TIFFs
	faxjob.Filename = filepath.Join(os.TempDir(), "gofaxsend_"+faxjob.UUID.String()+".tif")
	defer os.Remove(faxjob.Filename)
	if err = faxfile.WriteTo(faxjob.Filename); err != nil {
		return
	}

	// Start communication session and open logfile
	sessionlog, err := gofaxlib.NewSessionLogger(jobid)
	if err != nil {
		return
	}
	qf.Set("commid", sessionlog.CommID())
	logger.Logger.Println("Logging events for commid", sessionlog.CommID(), "to", sessionlog.Logfile())
	sessionlog.Logf("Processing HylaFAX Job %d as %v", jobid, faxjob.UUID)

	// Query DynamicConfig
	if dcCmd := gofaxlib.Config.Gofaxsend.DynamicConfig; dcCmd != "" {
		sessionlog.Log("Calling DynamicConfig script", dcCmd)
		dc, err := gofaxlib.DynamicConfig(dcCmd, *deviceID, qf.GetString("owner"), qf.GetString("number"), fmt.Sprint(jobid))
		if err != nil {
			errmsg := fmt.Sprintln("Error calling DynamicConfig:", err)
			sessionlog.Log(errmsg)
			qf.Set("returned", strconv.Itoa(sendRetry))
			qf.Set("status", errmsg)
			if err = qf.Write(); err != nil {
				sessionlog.Logf("Error updating qfile:", err)
			}
			// Retry, as this is an internal error executing the DynamicConfig script which could recover later
			return sendRetry, nil
		}

		// Check if call should be rejected
		if gofaxlib.DynamicConfigBool(dc.GetString("RejectCall")) {
			errmsg := "Transmission rejected by DynamicConfig"
			sessionlog.Log(errmsg)
			qf.Set("returned", strconv.Itoa(sendFailed))
			qf.Set("status", errmsg)
			if err = qf.Write(); err != nil {
				sessionlog.Logf("Error updating qfile:", err)
			}
			return sendFailed, nil
		}

		// Check if a custom identifier should be set
		if dynamicTsi := dc.GetString("LocalIdentifier"); dynamicTsi != "" {
			faxjob.Ident = dynamicTsi
		}

		if tagline := dc.GetString("TagLine"); tagline != "" {
			faxjob.Header = tagline
		}

		if prefix := dc.GetString("CallPrefix"); prefix != "" {
			faxjob.Number = fmt.Sprint(prefix, qf.GetString("external"))
		}

		if faxnumber := dc.GetString("FAXNumber"); faxnumber != "" {
			faxjob.Cidnum = faxnumber
		}

		if gatewayString := dc.GetString("Gateway"); gatewayString != "" {
			faxjob.Gateways = strings.Split(gatewayString, ",")
		}

	}

	switch gofaxlib.Config.Gofaxsend.CidName {
	case "sender":
		faxjob.Cidname = qf.GetString("sender")
	case "number":
		faxjob.Cidname = qf.GetString("number")
	case "cidnum":
		faxjob.Cidname = faxjob.Cidnum
	default:
		faxjob.Cidname = gofaxlib.Config.Gofaxsend.CidName
	}

	// Total attempted calls
	totdials, _ := qf.GetInt("totdials")
	// Consecutive failed attempts to place a call
	ndials, _ := qf.GetInt("ndials")
	// Total answered calls
	tottries, _ := qf.GetInt("tottries")

	//Auto fallback to slow baudrate after to many tries
	v17retry, err := strconv.Atoi(gofaxlib.Config.Gofaxsend.DisableV17AfterRetry)
	if err != nil {
		v17retry = 0
	}
	if v17retry > 0 && tottries >= v17retry {
		faxjob.DisableV17 = true
	}

	//Auto disable ECM after to many tries
	ecmretry, err := strconv.Atoi(gofaxlib.Config.Gofaxsend.DisableECMAfterRetry)
	if err != nil {
		ecmretry = 0
	}
	if ecmretry > 0 && tottries >= ecmretry {
		faxjob.UseECM = false
	}

	// Update status
	qf.Set("status", "Dialing")
	totdials++
	qf.Set("totdials", strconv.Itoa(totdials))
	if err = qf.Write(); err != nil {
		sessionlog.Log("Error updating qfile:", err)
		return sendFailed, nil
	}
	// Default: Retry when transmission fails
	returned = sendRetry

	// Start transmission goroutine
	t := transmit(*faxjob, sessionlog)
	var result *gofaxlib.FaxResult
	var status string

	// Wait for events
StatusLoop:
	for {
		select {
		case page := <-t.PageSent():
			qf.Set("npages", strconv.Itoa(int(page.Page)))
			qf.Set("dataformat", page.EncodingName)
			if err = qf.Write(); err != nil {
				sessionlog.Log("Error updating qfile:", err)
			}

		case result = <-t.Result():
			qf.Set("signalrate", strconv.Itoa(int(result.TransferRate)))
			qf.Set("csi", result.RemoteID)

			// Break if call is hung up
			if result.Hangupcause != "" {
				// Fax Finished
				status = result.ResultText
				if result.Success {
					returned = sendDone
				}
				break StatusLoop
			}

			// Negotiation finished
			negstatus := fmt.Sprint("Sending ", result.TransferRate)
			if result.Ecm {
				negstatus = negstatus + "/ECM"
			}
			status = negstatus
			tottries++
			ndials = 0
			qf.Set("status", status)
			qf.Set("tottries", strconv.Itoa(tottries))
			qf.Set("ndials", strconv.Itoa(ndials))
			if err = qf.Write(); err != nil {
				sessionlog.Log("Error updating qfile:", err)
			}

		case faxerr := <-t.Errors():
			ndials++
			qf.Set("ndials", strconv.Itoa(ndials))
			status = faxerr.Error()
			if faxerr.Retry() {
				returned = sendRetry
			}
			break StatusLoop
		}
	}

	qf.Set("status", status)
	qf.Set("returned", strconv.Itoa(returned))
	if err = qf.Write(); err != nil {
		sessionlog.Log("Error updating qfile:", err)
	}

	xfl := &gofaxlib.XFRecord{}
	xfl.Commid = sessionlog.CommID()
	xfl.Modem = *deviceID
	xfl.Jobid = uint(jobid)
	xfl.Jobtag = qf.GetString("jobtag")
	xfl.Sender = qf.GetString("mailaddr")
	xfl.Destnum = qf.GetString("number")
	xfl.Owner = qf.GetString("owner")

	if result != nil {
		if result.Success {
			sessionlog.Logf("Fax sent successfully. Hangup Cause: %v. Result: %v", result.Hangupcause, status)
		} else {
			sessionlog.Logf("Fax failed. Retry: %v. Hangup Cause: %v. Result: %v", returned == sendRetry, result.Hangupcause, status)
		}
		xfl.SetResult(result)
	} else {
		sessionlog.Logf("Call failed. Retry: %v. Result: %v", returned == sendRetry, status)
		xfl.Reason = status
	}

	if err = xfl.SaveTransmissionReport(); err != nil {
		sessionlog.Log(err)
	}

	return returned, nil
}
