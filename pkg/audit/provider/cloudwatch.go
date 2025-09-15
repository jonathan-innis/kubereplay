package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cloudwatchlogstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	auditmodel "github.com/joinnis/kubereplay/pkg/audit/model"
	"github.com/joinnis/kubereplay/pkg/object"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/types"
)

type CloudWatch struct {
	client       *cloudwatchlogs.Client
	logGroupName string
}

func NewCloudWatch(logGroupName, region string) (*CloudWatch, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	if region != "" {
		cfg.Region = region
	}

	return &CloudWatch{
		client:       cloudwatchlogs.NewFromConfig(cfg),
		logGroupName: logGroupName,
	}, nil
}

func (c *CloudWatch) Query(ctx context.Context, query string, startTime, endTime time.Time) (cloudwatchlogs.GetQueryResultsOutput, error) {
	startQuery, err := c.client.StartQuery(ctx, &cloudwatchlogs.StartQueryInput{
		QueryString:         lo.ToPtr(query),
		StartTime:           lo.ToPtr(startTime.Unix()),
		EndTime:             lo.ToPtr(endTime.Unix()),
		LogGroupIdentifiers: []string{c.logGroupName},
	})
	if err != nil {
		return cloudwatchlogs.GetQueryResultsOutput{}, err
	}
	var result *cloudwatchlogs.GetQueryResultsOutput
	lo.WaitFor(func(_ int) bool {
		result, err = c.client.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{
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

func (c *CloudWatch) GetEvents(ctx context.Context, parser object.ObjectParser, cmdType string, startTime, endTime time.Time, nn types.NamespacedName) ([]auditmodel.Event, error) {
	var query string
	switch cmdType {
	case "get":
		query = parser.GetQuery(nn)
	case "describe":
		query = parser.DescribeQuery(nn)
	default:
		panic(fmt.Sprintf("invalid command type: %s", cmdType))
	}

	res, err := c.Query(ctx, query, startTime, endTime)
	if err != nil {
		return nil, err
	}
	var auditEvents []auditmodel.Event
	for _, r := range res.Results {
		for _, field := range r {
			if *field.Field == "@message" {
				var auditEvent auditmodel.Event
				lo.Must0(json.Unmarshal([]byte(*field.Value), &auditEvent))
				auditEvents = append(auditEvents, auditEvent)
			}
		}
	}
	return auditEvents, nil
}
