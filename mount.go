package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	zfs "github.com/bakins/go-zfs"
	"github.com/spf13/cobra"
)

var mountDeviceCmd = &cobra.Command{
	Use:   "mountdevice <mount dir> <mount device> <json options>",
	Short: "Mount device mounts the device to a global path which individual pods can then bind mount.",
	Run:   runMountDeviceCmd,
}

var unmountDeviceCmd = &cobra.Command{
	Use:   "unmountdevice <mount device>",
	Short: "Unmounts the global mount for the device. ",
	Run:   runUnmountDeviceCmd,
}

var mountCmd = &cobra.Command{
	Use:   "mount <mount dir> <json options>",
	Short: "Mount the volume at the mount dir.",
	Run:   runMountCmd,
}

var unmountCmd = &cobra.Command{
	Use:   "unmount <mount dir>",
	Short: "Unmount the volume.",
	Run:   runUnmountCmd,
}

func runUnmountDeviceCmd(cmd *cobra.Command, args []string) {
	r := result{}
	if len(args) < 1 {
		r.emit(-2, cmd.Use)
		return
	}

	msg, err := unmount(args[0])
	if err != nil {
		r.error(err)
	}
	r.emit(0, msg)
}

func runMountDeviceCmd(cmd *cobra.Command, args []string) {
	r := result{}
	if len(args) < 3 {
		r.emit(-2, cmd.Use)
		return
	}

	mountpoint := args[0]

	var p params
	if err := json.Unmarshal([]byte(args[2]), &p); err != nil {
		r.emit(-1, fmt.Sprintf("unable to unmarshal json: %v", err))
		return
	}

	if p.Dataset == "" {
		r.emit(-2, "dataset is required")
		return
	}

	r.Device = filepath.Join(getParent(), p.Dataset)

	m, err := isMounted(r.Device, mountpoint)
	if err != nil {
		r.emit(-4, fmt.Sprintf("unable to get mounts: %v", err))
		return
	}

	if m {
		r.emit(0, "already mounted")
		return
	}

	// need to get current mountpoint and bind mount.
	// this is so we can leve it mounted in the "normal" zfs dir
	// as multiple containers/pods may mount for some reason
	ds, err := zfs.GetDataset(r.Device)
	if err != nil {
		r.emit(-4, fmt.Sprintf("failed to get dataset: %v", err))
		return
	}

	if ds.Type != "filesystem" {
		r.emit(-3, "existing dataset is not a filesystem")
		return
	}

	if ds.Mountpoint == mountpoint {
		// mountpoint is set, but is unmounted, so try to mount
		if _, err := ds.Mount(false, nil); err != nil {
			r.emit(-3, fmt.Sprintf("failed to mount dataset: %v", err))
			return
		}
		r.emit(0, "")
		return
	}

	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		r.emit(-3, fmt.Sprintf("failed to create mount point: %v", err))
		return
	}

	command := exec.Command("mount", "-o", "bind", ds.Mountpoint, mountpoint)
	if out, err := command.CombinedOutput(); err != nil {
		r.emit(-3, fmt.Sprintf("failed to mount dataset: %v: %s", err, string(out)))
		return
	}
	r.emit(0, "")

}

func runMountCmd(cmd *cobra.Command, args []string) {
	r := result{}
	if len(args) < 2 {
		r.emit(-2, cmd.Use)
		return
	}

	mountpoint := args[0]

	var p params
	if err := json.Unmarshal([]byte(args[1]), &p); err != nil {
		r.emit(-1, fmt.Sprintf("unable to unmarshal json: %v", err))
		return
	}

	if p.Dataset == "" {
		r.emit(-2, "dataset is required")
		return
	}

	r.Device = filepath.Join(getParent(), p.Dataset)

	m, err := isMounted(r.Device, mountpoint)
	if err != nil {
		r.emit(-4, fmt.Sprintf("unable to get mounts: %v", err))
		return
	}

	if m {
		r.emit(0, "already mounted")
		return
	}

	// need to get current mountpoint and bind mount.
	// this is so we can leve it mounted in the "normal" zfs dir
	// as multiple containers/pods may mount for some reason
	ds, err := zfs.GetDataset(r.Device)
	if err != nil {
		r.emit(-4, fmt.Sprintf("failed to get dataset: %v", err))
		return
	}

	if ds.Type != "filesystem" {
		r.emit(-3, "existing dataset is not a filesystem")
		return
	}

	if ds.Mountpoint == mountpoint {
		// mountpoint is set, but is unmounted, so try to mount
		if _, err := ds.Mount(false, nil); err != nil {
			r.emit(-3, fmt.Sprintf("failed to mount dataset: %v", err))
			return
		}
		r.emit(0, "")
		return
	}

	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		r.emit(-3, fmt.Sprintf("failed to create mount point: %v", err))
		return
	}

	command := exec.Command("mount", "-o", "bind", ds.Mountpoint, mountpoint)
	if out, err := command.CombinedOutput(); err != nil {
		r.emit(-3, fmt.Sprintf("failed to mount dataset: %v: %s", err, string(out)))
		return
	}
	r.emit(0, "")

}

func runUnmountCmd(cmd *cobra.Command, args []string) {
	r := result{}
	if len(args) < 1 {
		r.emit(-2, cmd.Use)
		return
	}

	msg, err := unmount(args[0])
	if err != nil {
		r.error(err)
	}
	r.emit(0, msg)
}

func unmount(mountpoint string) (string, error) {

	mounts, err := parseMounts()
	if err != nil {
		return "", err
	}

	found := false
	for _, m := range mounts {
		if m.mountpoint == mountpoint {
			found = true
			break
		}
	}
	if !found {
		return "already unmounted", nil
	}

	command := exec.Command("umount", mountpoint)
	if _, err := command.Output(); err != nil {
		return "", err
	}

	return "", nil
}

func init() {
	rootCmd.AddCommand(mountCmd)
	rootCmd.AddCommand(mountDeviceCmd)
	rootCmd.AddCommand(unmountCmd)
	rootCmd.AddCommand(unmountDeviceCmd)

}
