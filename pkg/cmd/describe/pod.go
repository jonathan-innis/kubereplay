package describe

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var podCmd = &cobra.Command{
	Use:   "pod <pod-name>",
	Short: "Get audit log events for a pod",
	Long: `Get audit log events for a specific pod from Kubernetes audit logs.

This command analyzes audit logs to extract key pod lifecycle events including:
  - Pod creation
  - Node binding (shows which node and when)
  - Karpenter nominations
  - Status updates and phase changes

Data Sources:
  Use either --audit-log for local files or --log-group for AWS CloudWatch Logs.
  Exactly one must be specified.

Examples:
  # Analyze pod from local audit log
  kubereplay describe pod nginx-pod -n default -f /var/log/audit.log

  # Analyze pod from CloudWatch (requires AWS credentials)
  kubereplay describe pod nginx-pod -n kube-system -g /aws/eks/prod-cluster/audit -r us-west-2

Output includes timestamps, event types, descriptions, and node information where applicable.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		podName := args[0]
		namespace, _ := cmd.Flags().GetString("namespace")
		auditLogPath, _ := cmd.Flags().GetString("audit-log")
		logGroup, _ := cmd.Flags().GetString("log-group")
		region, _ := cmd.Flags().GetString("region")
		start, _ := cmd.Flags().GetDuration("start")
		end, _ := cmd.Flags().GetDuration("end")

		if auditLogPath == "" && logGroup == "" {
			fmt.Println("Error: Either --audit-log or --log-group must be specified")
			return
		}

		if auditLogPath != "" && logGroup != "" {
			fmt.Println("Error: Cannot specify both --audit-log and --log-group")
			return
		}
		startTime := time.Now().Add(-start)
		endTime := time.Now().Add(-end)

		if err := RunDescribe(ctx, cmd, startTime, endTime, podName, namespace, auditLogPath, logGroup, region); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	},
}

func init() {
	Cmd.AddCommand(podCmd)
	podCmd.Flags().StringP("namespace", "n", "default", "Namespace of the pod")
	podCmd.Flags().StringP("audit-log", "f", "", "Path to audit log file")
	podCmd.Flags().StringP("log-group", "g", "", "AWS CloudWatch log group name")
	podCmd.Flags().StringP("region", "r", "", "AWS region for CloudWatch log group")
	podCmd.Flags().DurationP("start", "", time.Hour*24, "Start time for log parsing in time.Duration string format")
	podCmd.Flags().DurationP("end", "", 0, "End time for log parsing in time.Duration string format")
}
