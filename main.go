package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	zfs "github.com/bakins/go-zfs"
	units "github.com/docker/go-units"
)

type result struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Device  string `json:"device,omitempty"`
}

type params struct {
	Dataset     string `json:"dataset"`
	Compression string `json:"compression"`
	Quota       string `json:"quota"`
	Reservation string `json:"reservation"`
}

func usage() {
	cmd := os.Args[0]
	fmt.Printf(`
Usage:
%s init
%s attach <json params>
%s detach <mount device>
%s mount <mount dir> <mount device> <json params>
%s unmount <mount dir>
`, cmd, cmd, cmd, cmd, cmd)
	os.Exit(1)
}

func (r result) emit(rc int, message string) {
	out := os.Stderr
	switch rc {
	case 0:
		r.Status = "Success"
		out = os.Stdout
	default:
		r.Status = "Failed"
	}
	r.Message = message
	enc := json.NewEncoder(out)
	enc.Encode(&r)
	os.Exit(rc)
}

func main() {
	parent := flag.String("parent", "rpool/k8s/volumes", "parent dataset for volumes")
	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		usage()
	}

	action := args[0]
	args = args[1:]

	switch action {
	case "init":
		result{}.emit(0, "")
	case "attach":
		doAttach(*parent, args)
	case "detach":
		doDetach(*parent, args)
	case "mount":
		doMount(args)
	case "unmount":
		doUnmount(args)
	default:
		usage()
	}
}

func doUnmount(args []string) {
	r := result{}
	if len(args) < 1 {
		r.emit(-2, "mountpoint is required")
	}
	mountpoint := args[0]

	mounts, err := parseMounts()
	if err != nil {
		r.emit(-4, fmt.Sprintf("unable to get mounts: %v", err))
	}

	found := false
	for _, m := range mounts {
		if m.mountpoint == mountpoint {
			found = true
			break
		}
	}
	if !found {
		r.emit(0, "already unmounted")
	}

	cmd := exec.Command("umount", mountpoint)
	if _, err := cmd.Output(); err != nil {
		r.emit(-3, fmt.Sprintf("failed to unmount dataset: %v", err))
	}
	r.emit(0, "")
}

func doMount(args []string) {
	r := result{}
	if len(args) < 2 {
		r.emit(-2, "device and mountpoint are required")
	}

	mountpoint := args[0]
	r.Device = args[1]
	// XXX: we do not current use the options, but we could
	// infer some mount options here.

	m, err := isMounted(r.Device, mountpoint)
	if err != nil {
		r.emit(-4, fmt.Sprintf("unable to get mounts: %v", err))
	}

	if m {
		r.emit(0, "already mounted")
	}

	// need to get current mountpoint and bind mount.
	// this is so we can leve it mounted in the "normal" zfs dir
	// as multiple containers/pods may mount for some reason
	ds, err := zfs.GetDataset(r.Device)
	if err != nil {
		r.emit(-4, fmt.Sprintf("failed to get dataset: %v", err))
	}

	if ds.Type != "filesystem" {
		r.emit(-3, "existing dataset is not a filesystem")
	}

	if ds.Mountpoint == mountpoint {
		// mountpoint is set, but is unmounted, so try to mount
		if _, err := ds.Mount(false, nil); err != nil {
			r.emit(-3, fmt.Sprintf("failed to mount dataset: %v", err))
		}
		r.emit(0, "")
	}

	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		r.emit(-3, fmt.Sprintf("failed to create mount point: %v", err))
	}

	cmd := exec.Command("mount", "-o", "bind", ds.Mountpoint, mountpoint)
	if out, err := cmd.CombinedOutput(); err != nil {
		r.emit(-3, fmt.Sprintf("failed to mount dataset: %v: %s", err, string(out)))
	}
	r.emit(0, "")
}

func doDetach(parent string, args []string) {
	// XXX: we do not currenlty delete datasets. perhaps
	// set an option to do so
	result{}.emit(0, "")
}

func doAttach(parent string, args []string) {
	data := strings.Join(args, "")

	r := result{}

	var p params
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		r.emit(-1, fmt.Sprintf("unable to unmarshal json: %v", err))
	}

	if p.Dataset == "" {
		r.emit(-2, "dataset is required")
	}

	if p.Quota == "" {
		r.emit(-2, "quota is required")
	}

	if p.Reservation == "" {
		p.Reservation = "0"
	}

	quota, err := units.RAMInBytes(p.Quota)
	if err != nil {
		r.emit(-2, fmt.Sprintf("unable to parse quota: %v", err))
	}
	if quota <= 0 {
		r.emit(-2, "quota must be greater than 0")
	}

	reservation, err := units.RAMInBytes(p.Reservation)
	if err != nil {
		r.emit(-2, fmt.Sprintf("unable to parse reservation: %v", err))
	}

	if quota < reservation {
		r.emit(-2, "quota must be greater than or equal to reservation")
	}

	options := make(map[string]string)
	options["quota"] = strconv.FormatInt(quota, 10)
	if reservation > 0 {
		options["reservation"] = strconv.FormatInt(reservation, 10)
	}

	if p.Compression != "" {
		options["compression"] = p.Compression
	}

	r.Device = path.Join(parent, p.Dataset)
	// does the volume already exist?
	ds, err := zfs.GetDataset(r.Device)
	if err == nil {
		if ds.Type != "filesystem" {
			r.emit(-3, "existing dataset is not a filesystem")
		}
		// XXX: we will not alter existing volumes for now
		r.emit(0, "found existing filesystem")
	}

	// HACK: zfs package provides no way to check if it not found
	if !strings.Contains(err.Error(), "does not exist") {
		r.emit(-3, fmt.Sprintf("failed to check for filesystem: %v", err))
	}

	if _, err := zfs.CreateFilesystem(r.Device, options); err != nil {
		r.emit(-4, fmt.Sprintf("failed to create filesystem: %v", err))
	}

	r.emit(0, "created filesystem")

}
