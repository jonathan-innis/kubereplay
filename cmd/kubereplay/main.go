package main

import (
	"os"

	"github.com/joinnis/kubereplay/pkg/cmd/describe"
	"github.com/joinnis/kubereplay/pkg/cmd/get"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "kubereplay",
	Short: "kubectl plugin to analyze audit logs for Kubernetes objects",
	Long: `kubereplay extracts relevant state information for Kubernetes objects from audit logs.

It can parse audit logs from local files or AWS CloudWatch Logs to identify key events
such as pod creation, binding, Karpenter nominations, and status changes.

Examples:
  # Parse local audit log file
  kubereplay get pod my-pod -n default -f /path/to/audit.log

  # Parse from AWS CloudWatch Logs
  kubereplay get pod my-pod -n default -g /aws/eks/cluster-name/audit -r us-west-2
  
  # Get node YAML from audit logs
  kubereplay get node i-0871709ffb35ae35b -g /aws/eks/cluster-name/audit
  
  # Get node events from audit logs
  kubereplay describe node i-0871709ffb35ae35b -g /aws/eks/cluster-name/audit`,
}

func init() {
	root.AddCommand(describe.Cmd)
	root.AddCommand(get.Cmd)
}

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
