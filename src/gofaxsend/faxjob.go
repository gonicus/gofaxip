package main

import (
	"code.google.com/p/go-uuid/uuid"
)

// Fax job containing everything FreeSWITCH needs
type FaxJob struct {
	// FreeSWITCH Channel UUID (we generate this)
	UUID uuid.UUID
	// Destination number
	Number string
	// Caller ID number
	Cidnum string
	// Caller ID name
	Cidname string
	// TIFF file to send
	Filename string
	// Use ECM (default: true)
	UseECM bool
	// Fax ident
	Ident string
	// String for header (i.e. sender company name)
	// Page header with timestamp, header, ident, pageno will be added
	// if this Header is non empty
	Header string
}

func NewFaxJob() *FaxJob {
	return &FaxJob{
		UUID:   uuid.NewRandom(),
		UseECM: true,
	}
}
