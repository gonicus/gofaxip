package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)

const (
	// Access mode for newly created queue files.
	NEW_QFILE_MODE = 0660
)

type param struct {
	Tag   string
	Value string
}

type Qfile struct {
	filename string
	params   []param
}

func OpenQfile(filename string) (*Qfile, error) {
	var err error

	// Open queue file
	qfh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	q := new(Qfile)
	q.filename = filename

	// Lock queue file using flock (like Hylafax)
	err = syscall.Flock(int(qfh.Fd()), syscall.LOCK_EX)
	if err != nil {
		qfh.Close()
		return nil, err
	}

	// Read tags
	line := 1
	scanner := bufio.NewScanner(qfh)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			qfh.Close()
			return nil, errors.New(fmt.Sprintf("%s: Error parsing line %d", filename, line))
		}
		q.params = append(q.params, param{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])})
		line++
	}
	if err = scanner.Err(); err != nil {
		qfh.Close()
		return nil, err
	}

	return q, nil

}

func (q *Qfile) Write() error {
	qfh, err := os.OpenFile(q.filename, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	err = syscall.Flock(int(qfh.Fd()), syscall.LOCK_EX)
	if err != nil {
		qfh.Close()
		return err
	}

	// Truncate after acquiring lock!
	qfh.Truncate(0)

	for _, param := range q.params {
		if _, err := qfh.WriteString(fmt.Sprintf("%s:%s\n", param.Tag, param.Value)); err != nil {
			return err
		}
	}

	err = qfh.Close()
	if err != nil {
		return err
	}
	return nil
}

// Return slice containting all values for
// given tag.
func (q *Qfile) GetAll(tag string) []string {
	var result []string
	for _, param := range q.params {
		if param.Tag == tag {
			result = append(result, param.Value)
		}
	}
	return result
}

func (q *Qfile) GetFirst(tag string) string {
	for _, param := range q.params {
		if param.Tag == tag {
			return param.Value
		}
	}
	return ""
}

// Replace the value of the first found param
// with given value.
// If the param does not exist, append it.
func (q *Qfile) Set(tag string, value string) error {
	for i, param := range q.params {
		if param.Tag == tag {
			q.params[i].Value = value
			return nil
		}
	}
	return errors.New("Tag not found")
}

// Add a param with given tag and value. If the
// tag already exists, a second one is added.
func (q *Qfile) Add(tag string, value string) {
	q.params = append(q.params, param{tag, value})
}
