package get

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
	Use:   "get",
	Short: "Get Kubernetes resources from audit log events",
	Long: `Get Kubernetes resources from audit log events from local files or CloudWatch Logs.

Supported resources:
  pod    Get a specific pod
  node   Get a specific node

Additional Flags:
  --at           Exact time in RFC3339 time to get state for the resource
  --start        Duration value from the current time to start querying the audit logs
  --end          Duration value from the current time to finish querying the audit logs

Data sources:
  --audit-log    Local audit log file path
  --log-group    AWS CloudWatch log group name
  --region       AWS region for CloudWatch log group
  --account      AWS account ID for cross-account access

Examples:
  # Get pod from local file
  kubereplay get pod my-pod -n kube-system -f /var/log/audit.log

  # Get pod from CloudWatch
  kubereplay get pod my-pod -n default -g /aws/eks/my-cluster/audit -r us-west-2
  
  # Get node from Cloudwatch at time 2025-09-15T15:56:21
  kubereplay get node i-0123456789 -g /aws/eks/my-cluster/audit --at 2025-09-15T15:56:21`,
}

func RunGet(ctx context.Context, cmd *cobra.Command, startTime, endTime time.Time, nn types.NamespacedName, auditLogPath, logGroup, region string) error {
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
	fmt.Println(object.NewObjectParserFrom(cmd.Name()).Coalesce(nn, parsedEvents).Get())
	return nil
}
