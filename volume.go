package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var getVolumeNameCmd = &cobra.Command{
	Use:   "getvolumename <json options>",
	Short: "Get a cluster wide unique volume name for the volume. ",
	Run:   runGetVolumeNameCmd,
}

func runGetVolumeNameCmd(cmd *cobra.Command, args []string) {
	r := result{}
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

	hostname, err := os.Hostname()
	if err != nil {
		r.emit(-3, fmt.Sprintf("unable to determine node name: %v", err))
		return
	}

	r.VolumeName = strings.Join([]string{hostname, getParent(), p.Dataset}, ":")
	r.emit(0, "")
}

func init() {
	rootCmd.AddCommand(getVolumeNameCmd)
}
