package gofaxsend

import (
	"errors"
	"strconv"
)

// Qmemory provides the same functionality as Qfile, but all in-memory. It implements Qfiler.
type Qmemory struct {
	params map[string][]string

}

// NewQmemory instantiates a new Qmemory with the given parameters.
func NewQmemory(parameters map[string][]string) *Qmemory {
	if parameters == nil {
		parameters = make(map[string][]string)
	}
	return &Qmemory{
		params: parameters,
	}
}

func (q Qmemory) Write() error {
	// no-op
	return nil
}

func (q Qmemory) GetAll(tag string) []string {
	res, ok := q.params[tag]
	if !ok {
		return []string{}
	}

	return res
}

func (q Qmemory) GetString(tag string) string {
	res, ok := q.params[tag]
	if !ok {
		return ""
	}

	return res[0]
}

func (q Qmemory) GetInt(tag string) (int, error) {
	v := q.GetString(tag)
	if v == "" {
		return 0, errors.New("tag not found")
	}

	return strconv.Atoi(v)
}

func (q Qmemory) Set(tag, value string) {
	q.params[tag] = []string{value}
}

func (q Qmemory) Add(tag, value string) {
	existing, _ := q.params[tag]
	q.params[tag] = append(existing, value)
}

