package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kubereplay",
	Short: "kubectl plugin to analyze audit logs for Kubernetes objects",
	Long: `kubereplay extracts relevant state information for Kubernetes objects from audit logs.

It can parse audit logs from local files or AWS CloudWatch Logs to identify key events
such as pod creation, binding, Karpenter nominations, and status changes.

Examples:
  # Parse local audit log file
  kubereplay get pod my-pod -n default -f /path/to/audit.log

  # Parse from AWS CloudWatch Logs
  kubereplay get pod my-pod -n default -g /aws/eks/cluster-name/audit -r us-west-2`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(getCmd)
}
