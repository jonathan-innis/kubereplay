package describe

import (
	"context"
	"fmt"
	"time"

	"github.com/joinnis/kubereplay/pkg/audit/provider"
	"github.com/joinnis/kubereplay/pkg/object"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var Cmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe audit log events for Kubernetes resources",
	Long: `Describe audit log events for Kubernetes resources from local files or CloudWatch Logs.

Supported resources:
  pod    Describe events for a specific pod
  node   Describe events for a specific node

Additional Flags:
  --start        Duration value from the current time to start querying the audit logs
  --end          Duration value from the current time to finish querying the audit logs

Data sources:
  --audit-log    Local audit log file path
  --log-group    AWS CloudWatch log group name
  --region       AWS region for CloudWatch log group
  --account      AWS account ID for cross-account access

Examples:
  # Get pod events from local file
  kubereplay describe pod my-pod -n kube-system -f /var/log/audit.log

  # Get pod events from CloudWatch
  kubereplay describe pod my-pod -n default -g /aws/eks/my-cluster/audit -r us-west-2`,
}

func RunDescribe(ctx context.Context, cmd *cobra.Command, startTime, endTime time.Time, name, namespace, auditLogPath, logGroup, region string) error {
	nn := types.NamespacedName{Namespace: namespace, Name: name}

	var err error
	var auditProvider provider.Provider
	if logGroup != "" {
		auditProvider, err = provider.NewCloudWatch(logGroup, region)
		if err != nil {
			return fmt.Errorf("initializing cloudwatch provider, %w", err)
		}

	} else {
		auditProvider, err = provider.NewFile(auditLogPath)
		if err != nil {
			return fmt.Errorf("initializing file provider, %w", err)
		}
	}
	auditEvents, err := auditProvider.GetEvents(ctx, object.NewObjectParserFrom(cmd.Name()), cmd.Parent().Name(), startTime, endTime, nn)
	if err != nil {
		return fmt.Errorf("parsing events, %w", err)
	}
	parsedEvents := object.ParseEvents(auditEvents)
	if len(parsedEvents) == 0 {
		fmt.Printf("No events found for: %s\n", nn)
		return nil
	}
	fmt.Println(object.NewObjectParserFrom(cmd.Name()).Coalesce(nn, parsedEvents).Describe())
	return nil
}
