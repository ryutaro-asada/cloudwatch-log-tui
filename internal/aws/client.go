package aws

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
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
	NextToken     *string
	Ctx           context.Context
}
type LogEventInput struct {
	LogGroupName   string
	LogStreamNames []string
	StartTime      time.Time
	EndTime        time.Time
	FilterPattern  string
	OutputFile     string
	Ctx            context.Context
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
	LogEvents []cwlTypes.FilteredLogEvent
	NextToken *string
}

func (c *Client) SetLogStreamOutput(lsNames []string, lsLastEvent []string, lsFirstEvent []string) *LogStreamOutput {
	lsOutput := &LogStreamOutput{}
	for i := 0; i < len(lsNames); i++ {
		ls := cwlTypes.LogStream{
			LogStreamName:       aws.String(lsNames[i]),
			LastEventTimestamp:  aws.Int64(time.Now().Unix()),
			LastIngestionTime:   aws.Int64(time.Now().Unix()),
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
		params.LogGroupNamePattern = &input.FilterPattern
	}

	if input.NextToken != nil {
		params.NextToken = input.NextToken
	}

	res, err := c.cwl.DescribeLogGroups(input.Ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log groups: %w", err)
	}

	return &LogGroupOutput{
		LogGroups: res.LogGroups,
		NextToken: res.NextToken,
	}, nil
}

// GetLogStreams retrieves log streams for a given log group
func (c *Client) GetLogStreams(input *LogStreamInput) (*LogStreamOutput, error) {
	params := &cwl.DescribeLogStreamsInput{
		LogGroupName: aws.String(input.LogGroupName),
		Limit:        aws.Int32(MaxItemsInLayout),
		OrderBy:      cwlTypes.OrderByLastEventTime,
		Descending:   aws.Bool(true),
	}

	if input.NextToken != nil {
		params.NextToken = input.NextToken
	}

	res, err := c.cwl.DescribeLogStreams(input.Ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log streams: %w", err)
	}
	return &LogStreamOutput{
		LogStreams: res.LogStreams,
		NextToken:  res.NextToken,
	}, nil
}

// GetLogEvents retrieves log events for a given log group and stream
func (c *Client) GetLogEvents(input *LogEventInput) (*LogEventOutput, error) {
	params := &cwl.FilterLogEventsInput{
		LogGroupName: aws.String(input.LogGroupName),
		StartTime:    aws.Int64(input.StartTime.UnixMilli()),
		EndTime:      aws.Int64(input.EndTime.UnixMilli()),
		Limit:        aws.Int32(1000),
	}
	if len(input.LogStreamNames) > 0 {
		params.LogStreamNames = input.LogStreamNames
	}

	if input.FilterPattern != "" {
		params.FilterPattern = &input.FilterPattern
	}

	res, err := c.cwl.FilterLogEvents(input.Ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log events: %w", err)
	}
	return &LogEventOutput{
		LogEvents: res.Events,
		NextToken: res.NextToken,
	}, nil
}

func (c *Client) WriteLogEvents(input *LogEventInput) error {
	params := &cwl.FilterLogEventsInput{
		LogGroupName: aws.String(input.LogGroupName),
		StartTime:    aws.Int64(input.StartTime.UnixMilli()),
		EndTime:      aws.Int64(input.EndTime.UnixMilli()),
	}
	if len(input.LogStreamNames) > 0 {
		params.LogStreamNames = input.LogStreamNames
	}

	if input.FilterPattern != "" {
		params.FilterPattern = &input.FilterPattern
	}

	outputFile := "output.txt"
	if input.OutputFile != "" {
		outputFile = input.OutputFile
	}

	// Create and overwrite the output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	paginator := cwl.NewFilterLogEventsPaginator(c.cwl, params, func(o *cwl.FilterLogEventsPaginatorOptions) {
		o.Limit = 10000
	})

	for paginator.HasMorePages() {
		res, err := paginator.NextPage(input.Ctx)
		if err != nil {
			return fmt.Errorf("unable to get log events: %v", err)
		}

		for _, event := range res.Events {
			message := aws.ToString(event.Message)
			_, err := file.WriteString(message + "\n")
			if err != nil {
				return fmt.Errorf("failed to write log message: %v", err)
			}
		}
	}
	return nil
}
