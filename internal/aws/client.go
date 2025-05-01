package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// Client represents a CloudWatch Logs client
type Client struct {
	cwl *cloudwatchlogs.Client
}

// NewClient creates a new CloudWatch Logs client
func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	return &Client{
		cwl: cloudwatchlogs.NewFromConfig(cfg),
	}, nil
}

// GetLogGroups retrieves all log groups
func (c *Client) GetLogGroups(ctx context.Context) ([]types.LogGroup, error) {
	var logGroups []types.LogGroup
	var nextToken *string

	for {
		output, err := c.cwl.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe log groups: %w", err)
		}

		logGroups = append(logGroups, output.LogGroups...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return logGroups, nil
}

// GetLogStreams retrieves log streams for a given log group
func (c *Client) GetLogStreams(ctx context.Context, logGroupName string) ([]types.LogStream, error) {
	var logStreams []types.LogStream
	var nextToken *string

	for {
		output, err := c.cwl.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: &logGroupName,
			NextToken:    nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe log streams: %w", err)
		}

		logStreams = append(logStreams, output.LogStreams...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return logStreams, nil
}

// GetLogEvents retrieves log events for a given log group and stream
func (c *Client) GetLogEvents(ctx context.Context, logGroupName, logStreamName string, startTime, endTime *time.Time) ([]types.OutputLogEvent, error) {
	var logEvents []types.OutputLogEvent
	var nextToken *string

	input := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  &logGroupName,
		LogStreamName: &logStreamName,
	}

	if startTime != nil {
		startTimeMillis := startTime.UnixNano() / int64(time.Millisecond)
		input.StartTime = &startTimeMillis
	}
	if endTime != nil {
		endTimeMillis := endTime.UnixNano() / int64(time.Millisecond)
		input.EndTime = &endTimeMillis
	}

	for {
		input.NextToken = nextToken
		output, err := c.cwl.GetLogEvents(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get log events: %w", err)
		}

		logEvents = append(logEvents, output.Events...)

		if output.NextForwardToken == nil || (nextToken != nil && *output.NextForwardToken == *nextToken) {
			break
		}
		nextToken = output.NextForwardToken
	}

	return logEvents, nil
}
