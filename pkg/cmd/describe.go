package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/joinnis/kubereplay/pkg/audit"
	"github.com/joinnis/kubereplay/pkg/parser"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var getCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe audit log events for Kubernetes resources",
	Long: `Describe audit log events for Kubernetes resources from local files or CloudWatch Logs.

Supported resources:
  pod    Describe events for a specific pod

Data sources:
  --audit-log    Local audit log file path
  --log-group    AWS CloudWatch log group name
  --region       AWS region for CloudWatch log group
  --account      AWS account ID for cross-account access

Examples:
  # Get pod events from local file
  kubereplay get pod my-pod -n kube-system -f /var/log/audit.log

  # Get pod events from CloudWatch
  kubereplay get pod my-pod -n default -g /aws/eks/my-cluster/audit -r us-west-2

  # Get pod events from CloudWatch in different account
  kubereplay get pod my-pod -n default -g /aws/eks/my-cluster/audit -r us-west-2 -a 123456789012`,
}

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

  # Analyze pod from CloudWatch in different account
  kubereplay describe pod nginx-pod -n kube-system -g /aws/eks/prod-cluster/audit -r us-west-2 -a 123456789012

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

		if err := getPodEvents(ctx, start, end, podName, namespace, auditLogPath, logGroup, region); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	},
}

func init() {
	getCmd.AddCommand(podCmd)
	podCmd.Flags().StringP("namespace", "n", "default", "Namespace of the pod")
	podCmd.Flags().StringP("audit-log", "f", "", "Path to audit log file")
	podCmd.Flags().StringP("log-group", "g", "", "AWS CloudWatch log group name")
	podCmd.Flags().StringP("region", "r", "", "AWS region for CloudWatch log group")
	podCmd.Flags().DurationP("start", "", time.Hour*24, "Start time for log parsing in time.Duration string format")
	podCmd.Flags().DurationP("end", "", 0, "End time for log parsing in time.Duration string format")
}

func getPodEvents(ctx context.Context, start, end time.Duration, name, namespace, auditLogPath, logGroup, region string) error {
	nn := types.NamespacedName{Namespace: namespace, Name: name}

	var err error
	var auditProvider audit.Provider
	if logGroup != "" {
		auditProvider, err = audit.NewCloudWatchProvider(logGroup, region)
		if err != nil {
			return fmt.Errorf("initializing cloudwatch provider, %w", err)
		}

	} else {
		auditProvider, err = audit.NewFileProvider(auditLogPath)
		if err != nil {
			return fmt.Errorf("initializing file provider, %w", err)
		}
	}
	auditEvents, err := auditProvider.GetEvents(ctx, start, end, nn)
	if err != nil {
		return fmt.Errorf("parsing events, %w", err)
	}
	parsedEvents := parser.ParseEvents(auditEvents)
	if len(parsedEvents) == 0 {
		fmt.Printf("No events found for pod: %s\n", nn)
		return nil
	}
	fmt.Println(parser.Pod{}.Parse(nn, parsedEvents))
	return nil
}
