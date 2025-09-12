package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cloudwatchlogstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
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

type PodEventInfo struct {
	Name         string
	CreationTime time.Time
	BindTime     time.Time
	EvictionTime time.Time
	DeletionTime time.Time
}

func (e PodEventInfo) String() string {
	return fmt.Sprintf(`
%s
%s
CreationTime: %s
BindTime: %s
EvictionTime: %s
DeletionTime: %s
`,
		e.Name,
		strings.Repeat("-", len(e.Name)),
		lo.Ternary(e.CreationTime.IsZero(), "N/A", e.CreationTime.String()),
		lo.Ternary(e.BindTime.IsZero(), "N/A", e.BindTime.String()),
		lo.Ternary(e.EvictionTime.IsZero(), "N/A", e.EvictionTime.String()),
		lo.Ternary(e.DeletionTime.IsZero(), "N/A", e.DeletionTime.String()),
	)
}

func (p *CloudWatchProvider) Parse(ctx context.Context, start, end time.Duration, podName, namespace string) ([]PodEvent, error) {
	startTime := time.Now().Add(-start)
	endTime := time.Now().Add(-end)

	res, err := p.Query(ctx, GetPodQuery(podName), startTime, endTime)
	if err != nil {
		return nil, err
	}
	podEventInfo := PodEventInfo{Name: podName}
	queryResults := toPodQueryResults(res.Results)
	for _, result := range queryResults {
		switch {
		case result.Verb == "create" && strings.Contains(result.RequestURI, "binding"):
			podEventInfo.BindTime = result.Timestamp
		case result.Verb == "create" && strings.Contains(result.RequestURI, "eviction"):
			podEventInfo.EvictionTime = result.Timestamp
		case result.Verb == "create":
			podEventInfo.CreationTime = result.Timestamp
		case result.Verb == "delete":
			podEventInfo.DeletionTime = result.Timestamp
		}
	}
	fmt.Println(podEventInfo)
	return nil, nil
}

type PodQueryResult struct {
	Timestamp  time.Time
	RequestURI string
	Verb       string
}

func toPodQueryResults(results [][]cloudwatchlogstypes.ResultField) []PodQueryResult {
	return lop.Map(results, func(res []cloudwatchlogstypes.ResultField, _ int) PodQueryResult {
		queryResult := PodQueryResult{}
		for _, field := range res {
			switch *field.Field {
			case "@timestamp":
				queryResult.Timestamp = lo.Must(time.Parse("2006-01-02 15:04:05.999", *field.Value))
			case "verb":
				queryResult.Verb = *field.Value
			case "requestURI":
				queryResult.RequestURI = *field.Value
			}
		}
		return queryResult
	})
}

func GetPodQuery(podName string) string {
	const podQueryTemplate = `
fields @timestamp, requestURI, verb
| filter @logStream like "apiserver"
| filter verb like /create|delete/
| filter requestURI like "pods"
| filter @message like "%s"
`
	return fmt.Sprintf(podQueryTemplate, podName)
}
