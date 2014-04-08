package main

import (
	"errors"
	"fmt"
	"gofaxlib/logger"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

// A whole fax
type FaxFile struct {
	items []faxItem
}

// Parse a fax entry from queue file and add
// item to fax
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

// Combine input items and write output to given
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
