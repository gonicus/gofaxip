package gofaxlib

// Functions to parse and write
// HylaFax configuration files
//
// Right now this only contains a very simple
// implementation of DynamicConfig.
// The only supported option is "RejectCall: true"
//
// TODO: Merge this with qfile.go
// because it's pretty similar

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

type param struct {
	Tag   string
	Value string
}

type HylaConfig struct {
	params []param
}

func (h *HylaConfig) GetFirst(tag string) string {
	for _, param := range h.params {
		if param.Tag == tag {
			return param.Value
		}
	}
	return ""
}

func DynamicConfig(command string, cidnum string, cidname string, recipient string) (*HylaConfig, error) {
	if Config.Gofaxd.DynamicConfig == "" {
		return nil, errors.New("No DynamicConfig command provided")
	}

	cmd := exec.Command(Config.Gofaxd.DynamicConfig, cidnum, cidname, recipient)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	h := new(HylaConfig)

	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		h.params = append(h.params, param{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])})
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return h, nil
}
