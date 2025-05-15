package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/aws"
)

const MaxItemsInLayout int32 = 50

// Client represents a CloudWatch Logs client
type Client struct {
	cwl *cwl.Client
}

// Input types for log groups, streams, and events
type LogGroupInput struct {
	FilterPattern string
	NextToken     *string
	Ctx           context.Context
}
type LogStreamInput struct {
	LogGroupName  string
	PrefixPattern string
	NextToken     *string
	Ctx           context.Context
}
type LogEventInput struct {
	Ctx context.Context
}

// Output types for log groups, streams, and events
type LogGroupOutput struct {
	LogGroups []cwlTypes.LogGroup
	NextToken *string
}
type LogStreamOutput struct {
	LogStreams []cwlTypes.LogStream
	NextToken  *string
}
type LogEventOutput struct {
	LogEvents []cwlTypes.OutputLogEvent
	NextToken *string
}

func (c *Client) SetLogStreamOutput(lsNames []string, lsLastEvent []string, lsFirstEvent []string ) *LogStreamOutput {
	lsOutput := &LogStreamOutput{}
	for i := 0; i < len(lsNames); i++ {
		ls := cwlTypes.LogStream{
			LogStreamName: aws.String(lsNames[i]),
			LastEventTimestamp: aws.Int64(time.Now().Unix()),
			LastIngestionTime: aws.Int64(time.Now().Unix()),
			FirstEventTimestamp: aws.Int64(time.Now().Unix()),
		}
		lsOutput.LogStreams = append(lsOutput.LogStreams, ls)
	}
	return lsOutput
}

// NewClient creates a new CloudWatch Logs client
func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	return &Client{
		cwl: cwl.NewFromConfig(cfg),
	}, nil
}

// GetLogGroups retrieves all log groups
func (c *Client) GetLogGroups(input *LogGroupInput) (*LogGroupOutput, error) {
	params := &cwl.DescribeLogGroupsInput{
		Limit: aws.Int32(MaxItemsInLayout),
	}

	if input.FilterPattern != "" {
		params.LogGroupNamePrefix = &input.FilterPattern
	}

	if input.NextToken != nil {
		params.NextToken = input.NextToken
	}

	output, err := c.cwl.DescribeLogGroups(input.Ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log groups: %w", err)
	}

	return &LogGroupOutput{
		LogGroups: output.LogGroups,
		NextToken: output.NextToken,
	}, nil
}

// GetLogStreams retrieves log streams for a given log group
func (c *Client) GetLogStreams(input *LogStreamInput) (*LogStreamOutput, error) {
	params := &cwl.DescribeLogStreamsInput{
		LogGroupName: aws.String(input.LogGroupName),
		Limit:        aws.Int32(MaxItemsInLayout),
	}

	if input.PrefixPattern != "" {
		params.LogStreamNamePrefix = &input.PrefixPattern
	}

	if input.NextToken != nil {
		params.NextToken = input.NextToken
	}

	output, err := c.cwl.DescribeLogStreams(input.Ctx, params)

	if err != nil {
		return nil, fmt.Errorf("failed to describe log streams: %w", err)
	}
	return &LogStreamOutput{
		LogStreams: output.LogStreams,
		NextToken:  output.NextToken,
	}, nil
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
