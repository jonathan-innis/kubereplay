package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cloudwatchlogstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/types"
)

type CloudWatchProvider struct {
	client       *cloudwatchlogs.Client
	logGroupName string
}

func NewCloudWatchProvider(logGroupName, region string) (*CloudWatchProvider, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	if region != "" {
		cfg.Region = region
	}

	return &CloudWatchProvider{
		client:       cloudwatchlogs.NewFromConfig(cfg),
		logGroupName: logGroupName,
	}, nil
}

func (p *CloudWatchProvider) Query(ctx context.Context, query string, start, end time.Time) (cloudwatchlogs.GetQueryResultsOutput, error) {
	startQuery, err := p.client.StartQuery(ctx, &cloudwatchlogs.StartQueryInput{
		QueryString:         lo.ToPtr(query),
		StartTime:           lo.ToPtr(start.Unix()),
		EndTime:             lo.ToPtr(end.Unix()),
		LogGroupIdentifiers: []string{p.logGroupName},
	})
	if err != nil {
		return cloudwatchlogs.GetQueryResultsOutput{}, err
	}
	var result *cloudwatchlogs.GetQueryResultsOutput
	lo.WaitFor(func(_ int) bool {
		result, err = p.client.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{
			QueryId: startQuery.QueryId,
		})
		if err != nil {
			return true
		}
		return result.Status == cloudwatchlogstypes.QueryStatusComplete
	}, time.Minute*5, 500*time.Millisecond)
	if err != nil {
		return cloudwatchlogs.GetQueryResultsOutput{}, err
	}
	return *result, nil
}

func (p *CloudWatchProvider) GetEvents(ctx context.Context, start, end time.Duration, nn types.NamespacedName) ([]Event, error) {
	startTime := time.Now().Add(-start)
	endTime := time.Now().Add(-end)

	res, err := p.Query(ctx, GetPodQuery(nn), startTime, endTime)
	if err != nil {
		return nil, err
	}
	var auditEvents []Event
	for _, r := range res.Results {
		for _, field := range r {
			if *field.Field == "@message" {
				var auditEvent Event
				lo.Must0(json.Unmarshal([]byte(*field.Value), &auditEvent))
				auditEvents = append(auditEvents, auditEvent)
			}
		}
	}
	return auditEvents, nil
}

type PodQueryResult struct {
	Timestamp  time.Time
	RequestURI string
	Verb       string
}

func GetPodQuery(nn types.NamespacedName) string {
	const podQueryTemplate = `
fields @timestamp, @message
| filter @logStream like "apiserver"
| filter verb like /create|delete/
| filter requestURI like "pods"
| filter @message like "%s"
| filter requestURI like "%s"
`
	return fmt.Sprintf(podQueryTemplate, nn.Name, nn.Namespace)
}
