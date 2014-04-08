package main

type FaxError interface {
	error
	Retry() bool
}

type faxError struct {
	msg   string
	retry bool
}

func NewFaxError(msg string, retry bool) faxError {
	return faxError{msg, retry}
}

func (e faxError) Error() string {
	return e.msg
}

func (e faxError) Retry() bool {
	return e.retry
}
