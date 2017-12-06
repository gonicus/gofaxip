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

// HylaConfig holds a set of HylaFAX configuration parameters
type HylaConfig struct {
	params []param
}

// GetString returns the first Value found matching given Tag
func (h *HylaConfig) GetString(tag string) string {
	tag = strings.ToLower(tag)
	for _, param := range h.params {
		if param.Tag == tag {
			return param.Value
		}
	}
	return ""
}

// DynamicConfig executes the given command and parses the output compatible to HylaFAX' DynamicConfig
func DynamicConfig(command string, args ...string) (*HylaConfig, error) {

	if command == "" {
		return nil, errors.New("No DynamicConfig command provided")
	}

	cmd := exec.Command(command, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	h := &HylaConfig{}

	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		h.params = append(h.params, param{strings.ToLower(strings.TrimSpace(parts[0])), strings.TrimSpace(parts[1])})
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return h, nil
}

// DynamicConfigBool interprets a DynamicConfig string value as truth value
func DynamicConfigBool(value string) bool {
	switch strings.ToLower(value) {
	case "true", "1", "yes":
		return true
	}

	return false
}
