package cmd

import (
	"fmt"
	"os"

	"github.com/joinnis/kubereplay/pkg/audit"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get audit log events for Kubernetes resources",
	Long: `Get audit log events for Kubernetes resources from local files or CloudWatch Logs.

Supported resources:
  pod    Get events for a specific pod

Data sources:
  --audit-log    Local audit log file path
  --log-group    AWS CloudWatch log group name

Examples:
  # Get pod events from local file
  kubereplay get pod my-pod -n kube-system -f /var/log/audit.log

  # Get pod events from CloudWatch
  kubereplay get pod my-pod -n default -g /aws/eks/my-cluster/audit`,
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
  kubereplay get pod nginx-pod -n default -f /var/log/audit.log

  # Analyze pod from CloudWatch (requires AWS credentials)
  kubereplay get pod nginx-pod -n kube-system -g /aws/eks/prod-cluster/audit

Output includes timestamps, event types, descriptions, and node information where applicable.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		podName := args[0]
		namespace, _ := cmd.Flags().GetString("namespace")
		auditLogPath, _ := cmd.Flags().GetString("audit-log")
		logGroup, _ := cmd.Flags().GetString("log-group")

		if auditLogPath == "" && logGroup == "" {
			fmt.Println("Error: Either --audit-log or --log-group must be specified")
			return
		}

		if auditLogPath != "" && logGroup != "" {
			fmt.Println("Error: Cannot specify both --audit-log and --log-group")
			return
		}

		if err := getPodEvents(podName, namespace, auditLogPath, logGroup); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	},
}

func init() {
	getCmd.AddCommand(podCmd)
	podCmd.Flags().StringP("namespace", "n", "default", "Namespace of the pod")
	podCmd.Flags().StringP("audit-log", "f", "", "Path to audit log file")
	podCmd.Flags().StringP("log-group", "g", "", "AWS CloudWatch log group name")
}

func getPodEvents(podName, namespace, auditLogPath, logGroup string) error {
	var events []audit.PodEvent
	var err error

	if logGroup != "" {
		provider, err := audit.NewCloudWatchProvider(logGroup)
		if err != nil {
			return fmt.Errorf("failed to parse audit events: %w", err)
		}
		events, err = provider.Parse(podName, namespace)
		if err != nil {
			return fmt.Errorf("failed to parse audit events: %w", err)
		}
	} else {
		if _, err := os.Stat(auditLogPath); os.IsNotExist(err) {
			return fmt.Errorf("audit log file does not exist: %s", auditLogPath)
		}
		events, err = audit.ParsePodEvents(auditLogPath, podName, namespace)
		if err != nil {
			return fmt.Errorf("failed to parse audit events: %w", err)
		}
	}

	if len(events) == 0 {
		fmt.Printf("No events found for pod %s in namespace %s\n", podName, namespace)
		return nil
	}

	fmt.Printf("Events for pod %s in namespace %s:\n\n", podName, namespace)
	for _, event := range events {
		fmt.Printf("Time: %s\n", event.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("Event: %s\n", event.Event)
		fmt.Printf("Description: %s\n", event.Description)
		if event.Node != "" {
			fmt.Printf("Node: %s\n", event.Node)
		}
		fmt.Println("---")
	}

	return nil
}
