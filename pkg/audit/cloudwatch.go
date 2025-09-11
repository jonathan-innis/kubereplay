package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type CloudWatchProvider struct {
	client       *cloudwatchlogs.Client
	logGroupName string
}

func NewCloudWatchProvider(logGroupName string) (*CloudWatchProvider, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &CloudWatchProvider{
		client:       cloudwatchlogs.NewFromConfig(cfg),
		logGroupName: logGroupName,
	}, nil
}

func (p *CloudWatchProvider) Parse(podName, namespace string) ([]PodEvent, error) {
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  aws.String(p.logGroupName),
		StartTime:     aws.Int64(startTime.UnixMilli()),
		EndTime:       aws.Int64(endTime.UnixMilli()),
		FilterPattern: aws.String(fmt.Sprintf(`{ $.objectRef.name = "%s" && $.objectRef.namespace = "%s" }`, podName, namespace)),
	}

	var events []PodEvent
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(p.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to fetch logs: %w", err)
		}

		for _, logEvent := range output.Events {
			var auditEvent AuditEvent
			if err := json.Unmarshal([]byte(*logEvent.Message), &auditEvent); err != nil {
				continue
			}

			if !isPodRelated(auditEvent, podName, namespace) {
				continue
			}

			podEvent := extractPodEvent(auditEvent)
			if podEvent != nil {
				events = append(events, *podEvent)
			}
		}
	}

	return events, nil
}
