package main

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	zfs "github.com/bakins/go-zfs"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <json params> <nodename>",
	Short: "Attach the volume specified by the given spec on the given host.",
	Run:   runAttachCmd,
}

var isAttachedCmd = &cobra.Command{
	Use:   "isattached <json options> <node name>",
	Short: "Check the volume is attached on the node.",
	Run:   runIsAttached,
}

var waitForAttachCmd = &cobra.Command{
	Use:   "waitforattach <mount device> <json params>",
	Short: "Wait for the volume to be attached on the remote node",
	Run:   runWaitForAttachCmd,
}

var detachCmd = &cobra.Command{
	Use:   "detach <mount device> <node name>",
	Short: "Detach the volume from the Kubelet node.",
	Run:   runDetachCmd,
}

func runWaitForAttachCmd(cmd *cobra.Command, args []string) {
	r := result{}
	if len(args) < 2 {
		r.emit(-2, cmd.Use)
		return
	}

	var p params
	if err := json.Unmarshal([]byte(args[1]), &p); err != nil {
		r.emit(-1, fmt.Sprintf("unable to unmarshal json: %v", err))
		return
	}

	if p.Dataset == "" {
		r.emit(-2, "dataset is required")
		return
	}

	r.Device = path.Join(getParent(), p.Dataset)

	if r.Device != args[0] {
		r.emit(-6, fmt.Sprintf("devices do not match: %s != %s", args[0], r.Device))
	}

	// does the volume already exist?
	ds, err := zfs.GetDataset(r.Device)
	if err != nil {
		r.error(err)
		return
	}

	if ds.Type != "filesystem" {
		r.emit(-3, "existing dataset is not a filesystem")
		return
	}
	// XXX: we will not alter existing volumes for now
	r.emit(0, "found existing filesystem")
	return
}

func runAttachCmd(cmd *cobra.Command, args []string) {
	r := result{}

	// node is second argument, but could be optional?
	if len(args) < 1 {
		r.emit(-2, cmd.Use)
		return
	}

	var p params
	if err := json.Unmarshal([]byte(args[0]), &p); err != nil {
		r.emit(-1, fmt.Sprintf("unable to unmarshal json: %v", err))
		return
	}

	if p.Dataset == "" {
		r.emit(-2, "dataset is required")
		return
	}

	if p.Quota == "" {
		r.emit(-2, "quota is required")
		return
	}

	if p.Reservation == "" {
		p.Reservation = "0"
	}

	quota, err := units.RAMInBytes(p.Quota)
	if err != nil {
		r.emit(-2, fmt.Sprintf("unable to parse quota: %v", err))
		return
	}
	if quota <= 0 {
		r.emit(-2, "quota must be greater than 0")
		return
	}

	reservation, err := units.RAMInBytes(p.Reservation)
	if err != nil {
		r.emit(-2, fmt.Sprintf("unable to parse reservation: %v", err))
		return
	}

	if quota < reservation {
		r.emit(-2, "quota must be greater than or equal to reservation")
		return
	}

	options := make(map[string]string)
	options["quota"] = strconv.FormatInt(quota, 10)
	if reservation > 0 {
		options["reservation"] = strconv.FormatInt(reservation, 10)
	}

	if p.Compression != "" {
		options["compression"] = p.Compression
	}

	r.Device = path.Join(getParent(), p.Dataset)
	// does the volume already exist?
	ds, err := zfs.GetDataset(r.Device)
	if err == nil {
		if ds.Type != "filesystem" {
			r.emit(-3, "existing dataset is not a filesystem")
			return
		}
		// XXX: we will not alter existing volumes for now
		r.emit(0, "found existing filesystem")
		return
	}

	// HACK: zfs package provides no way to check if it not found
	if !strings.Contains(err.Error(), "does not exist") {
		r.emit(-3, fmt.Sprintf("failed to check for filesystem: %v", err))
		return
	}

	if _, err := zfs.CreateFilesystem(r.Device, options); err != nil {
		r.emit(-4, fmt.Sprintf("failed to create filesystem: %v", err))
		return
	}

	r.emit(0, "created filesystem")
}

func runDetachCmd(cmd *cobra.Command, args []string) {
	// XXX: we do not currenlty delete datasets. perhaps
	// set an option to do so
	result{}.emit(0, "")
}

func runIsAttached(cmd *cobra.Command, args []string) {
	r := result{Attached: true}
	// https://github.com/kubernetes/kubernetes/blob/master/examples/volumes/flexvolume/lvm
	// just returns true
	r.emit(0, "")
}

func init() {
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(isAttachedCmd)
	rootCmd.AddCommand(waitForAttachCmd)
	rootCmd.AddCommand(detachCmd)
}
