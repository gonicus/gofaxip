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

package gofaxsend

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gonicus/gofaxip/gofaxlib/logger"
)

// One item (TIFF file) of a fax
type faxItem struct {
	// First directory in the file to transmit
	startdir uint

	// HDLC subaddress (unused)
	subaddr string

	// Input file of this item
	filename string
}

// FaxFile represents a whole fax optionally consisting of multiple documents
type FaxFile struct {
	items []faxItem
}

// AddItem parses a fax entry from queue file and adds
// it to the fax
func (f *FaxFile) AddItem(entry string) error {
	parts := strings.SplitN(entry, ":", 3)
	if len(parts) != 3 {
		return errors.New("Error parsing fax file entry")
	}

	faxitem := faxItem{
		subaddr: parts[1],
	}

	if startdir, err := strconv.ParseUint(parts[0], 10, 0); err == nil {
		faxitem.startdir = uint(startdir)
	} else {
		return err
	}

	filename, err := filepath.Abs(parts[2])
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err = os.Stat(filename); err != nil {
		return err
	}

	faxitem.filename = filename
	f.items = append(f.items, faxitem)
	return nil
}

// WriteTo combines input items and writes output to given
// path using tiffcp
func (f *FaxFile) WriteTo(outfile string) error {
	if len(f.items) == 0 {
		return errors.New("No part files found to combine")
	}

	args := make([]string, 0, len(f.items)+2)
	args = append(args, "-x")
	for _, item := range f.items {
		// BUG(markus): If the filename contains ',' tiffcp will fail
		args = append(args, fmt.Sprintf("%s,%d,", item.filename, item.startdir))
	}
	args = append(args, outfile)

	cmd := exec.Command("tiffcp", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Logger.Print(string(output))
		return err
	}

	return nil
}
