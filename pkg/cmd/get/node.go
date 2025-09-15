package get

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var nodeCmd = &cobra.Command{
	Use:   "node <node-name>",
	Short: "Get audit log events for a node",
	Long: `Get audit log events for a specific node from Kubernetes audit logs.

Data Sources:
  Use either --audit-log for local files or --log-group for AWS CloudWatch Logs.
  Exactly one must be specified.

Examples:
  # Get pod from local audit log
  kubereplay get node i-123456789 -f /var/log/audit.log

  # Get pod from CloudWatch (requires AWS credentials)
  kubereplay get node i-123456789 -g /aws/eks/prod-cluster/audit -r us-west-2

Output includes timestamps, event types, descriptions, and node information where applicable.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		name := args[0]
		auditLogPath, _ := cmd.Flags().GetString("audit-log")
		logGroup, _ := cmd.Flags().GetString("log-group")
		region, _ := cmd.Flags().GetString("region")
		start, _ := cmd.Flags().GetDuration("start")
		end, _ := cmd.Flags().GetDuration("end")
		at, _ := cmd.Flags().GetString("at")

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
		if at != "" {
			endTime = lo.Must(time.Parse(time.RFC3339, at))
		}

		if err := RunGet(ctx, cmd, startTime, endTime, types.NamespacedName{Name: name}, auditLogPath, logGroup, region); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	},
}

func init() {
	Cmd.AddCommand(nodeCmd)
	nodeCmd.Flags().StringP("audit-log", "f", "", "Path to audit log file")
	nodeCmd.Flags().StringP("log-group", "g", "", "AWS CloudWatch log group name")
	nodeCmd.Flags().StringP("region", "r", "", "AWS region for CloudWatch log group")
	nodeCmd.Flags().DurationP("start", "", time.Hour*24, "Start time for log parsing in time.Duration string format")
	nodeCmd.Flags().DurationP("end", "", 0, "End time for log parsing in time.Duration string format")
	nodeCmd.Flags().StringP("at", "", "", "Time to query the object state")
}
