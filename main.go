package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

type result struct {
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	Device     string `json:"device,omitempty"`
	VolumeName string `json:"volumeName,omitempty"`
	Attached   bool   `json:"attached,omitempty"`
}

type params struct {
	Dataset     string `json:"dataset"`
	Compression string `json:"compression"`
	Quota       string `json:"quota"`
	Reservation string `json:"reservation"`
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

func (r result) error(err error) {
	out := os.Stderr
	r.Status = "Failed"
	r.Message = err.Error()
	enc := json.NewEncoder(out)
	enc.Encode(&r)
	os.Exit(-1)
}

func helpCmd(cmd *cobra.Command, args []string) {
	cmd.Help()
}

var rootCmd = &cobra.Command{
	Use:   "zfs-flex-volume",
	Short: "Kubernetes Flex Volume Driver for ZFS",
	Run:   helpCmd,
}

func init() {
	rootCmd.PersistentFlags().StringP("parent", "", "k8s/volumes", "parent ZFS dataset")
}

func getParent() string {
	return rootCmd.PersistentFlags().Lookup("parent").Value.String()
}

func notSupported(cmd *cobra.Command, args []string) {
	fmt.Println(`{ "status": "Not supported" }`)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("root command failed: %v", err)
	}
}
