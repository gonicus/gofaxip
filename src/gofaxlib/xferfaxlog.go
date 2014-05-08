package gofaxlib

import (
	"fmt"
	"time"
)

const (
	// 19 fields
	XLOG_FORMAT = "%s\t%s\t%s\t%s\t%v\t%s\t%s\t\"%s\"\t\"%s\"\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\"%s\""
	TS_LAYOUT   = "01/02/06 15:04"
)

type XFRecord struct {
	Ts       time.Time
	Commid   string
	Modem    string
	Jobid    uint
	Jobtag   string
	Filename string
	Sender   string
	Destnum  string
	RemoteID string
	Params   uint
	Pages    uint
	Jobtime  time.Duration
	Conntime time.Duration
	Reason   string
	Cidname  string
	Cidnum   string
	Owner    string
	Dcs      string
}

func NewXFRecord(result *FaxResult) *XFRecord {
	duration := result.EndTs.Sub(result.StartTs)

	r := &XFRecord{
		Ts:       result.StartTs,
		Commid:   result.sessionlog.CommId(),
		RemoteID: result.RemoteID,
		Params:   EncodeParams(result.TransferRate, result.Ecm),
		Pages:    result.TransferredPages,
		Jobtime:  duration,
		Conntime: duration,
		Reason:   result.ResultText,
	}

	if len(result.PageResults) > 0 {
		r.Dcs = result.PageResults[0].EncodingName
	}

	return r
}

func (r *XFRecord) FormatTransmissionReport() string {
	return fmt.Sprintf(XLOG_FORMAT, r.Ts.Format(TS_LAYOUT), "SEND", r.Commid, r.Modem,
		r.Jobid, r.Jobtag, r.Sender, r.Destnum, r.RemoteID, r.Params, r.Pages,
		FormatDuration(r.Jobtime), FormatDuration(r.Conntime), r.Reason, "", "", "", r.Owner, r.Dcs)
}

func (r *XFRecord) FormatReceptionReport() string {
	return fmt.Sprintf(XLOG_FORMAT, r.Ts.Format(TS_LAYOUT), "RECV", r.Commid, r.Modem,
		r.Filename, "", "fax", r.Destnum, r.RemoteID, r.Params, r.Pages,
		FormatDuration(r.Jobtime), FormatDuration(r.Conntime), r.Reason,
		fmt.Sprintf("\"%s\"", r.Cidname), fmt.Sprintf("\"%s\"", r.Cidnum), "", "", r.Dcs)
}

func (r *XFRecord) SaveTransmissionReport() error {
	if Config.Hylafax.Xferfaxlog == "" {
		return nil
	}
	return AppendTo(Config.Hylafax.Xferfaxlog, r.FormatTransmissionReport())
}

func (r *XFRecord) SaveReceptionReport() error {
	if Config.Hylafax.Xferfaxlog == "" {
		return nil
	}
	return AppendTo(Config.Hylafax.Xferfaxlog, r.FormatReceptionReport())
}

func FormatDuration(d time.Duration) string {
	s := uint(d.Seconds())

	hours := s / (60 * 60)
	minutes := (s / 60) - (60 * hours)
	seconds := s % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// This only encodes bitrate and ECM use right now
func EncodeParams(baudrate uint, ecm bool) uint {

	var br uint
	switch {
	case baudrate > 12000:
		br = 5
	case baudrate > 9600:
		br = 4
	case baudrate > 7200:
		br = 3
	case baudrate > 4800:
		br = 2
	case baudrate > 2400:
		br = 1
	}

	var ec uint
	if ecm {
		ec = 1
	}

	return (br << 3) | (ec << 16)
}
