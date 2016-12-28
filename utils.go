package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type (
	mount struct {
		device     string
		mountpoint string
		filesystem string
	}
)

// parse mounts into an array of mounts
// only works on linux
func parseMounts() ([]*mount, error) {
	in, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer in.Close()

	var mounts []*mount
	s := bufio.NewScanner(in)

	for s.Scan() {
		parts := strings.Split(s.Text(), " ")
		if len(parts) < 3 {
			continue
		}
		mounts = append(mounts, &mount{device: parts[0], mountpoint: parts[1], filesystem: parts[2]})
	}

	if err := s.Err(); err != nil {
		return nil, err
	}
	return mounts, nil
}

func isMounted(device, mountpoint string) (bool, error) {
	mounts, err := parseMounts()
	if err != nil {
		return false, err
	}

	for _, m := range mounts {

		if m.device == device && m.mountpoint == mountpoint {
			if m.filesystem != "zfs" {
				return false, fmt.Errorf("unexpected filesystem: %s", m.filesystem)
			}
			return true, nil
		}
	}

	return false, nil
}
