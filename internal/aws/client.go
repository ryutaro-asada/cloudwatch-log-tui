// Package aws provides AWS CloudWatch Logs client functionality for the TUI application.
// It handles all interactions with AWS CloudWatch Logs API.
package aws

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

const (
	// MaxItemsInLayout defines the maximum number of items to display per page in the UI layout.
	MaxItemsInLayout int32 = 50
	// MaxLogEvents is the maximum number of events GetLogEvents returns.
	MaxLogEvents = 1000
	// MaxLogStreamNames is the FilterLogEvents limit for logStreamNames.
	MaxLogStreamNames = 100
	// DefaultOutputFile is the file WriteLogEvents writes to when no file is given.
	DefaultOutputFile = "output.txt"

	maxLogGroupNamePatternLength = 512
)

// See LogGroupNamePattern in
// https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_DescribeLogGroups.html
var logGroupNamePatternRegexp = regexp.MustCompile(`^[\.\-_/#A-Za-z0-9]*$`)

// Client represents a CloudWatch Logs client
type Client struct {
	cwl *cwl.Client
}

// LogGroupInput holds the parameters for GetLogGroups.
type LogGroupInput struct {
	FilterPattern string
	NextToken     *string
	Ctx           context.Context
}

// LogStreamInput holds the parameters for GetLogStreams.
type LogStreamInput struct {
	LogGroupName string
	NextToken    *string
	Ctx          context.Context
}

// LogEventInput holds the parameters for GetLogEvents and WriteLogEvents.
// EndTime is inclusive.
type LogEventInput struct {
	LogGroupName   string
	LogStreamNames []string
	StartTime      time.Time
	EndTime        time.Time
	FilterPattern  string
	OutputFile     string
	Ctx            context.Context
}

// LogGroupOutput holds one page of log groups.
type LogGroupOutput struct {
	LogGroups []cwlTypes.LogGroup
	NextToken *string
}

// LogStreamOutput holds one page of log streams.
type LogStreamOutput struct {
	LogStreams []cwlTypes.LogStream
	NextToken  *string
}

// LogEventOutput holds the events returned by GetLogEvents. NextToken is set
// when more events exist than were returned.
type LogEventOutput struct {
	LogEvents []cwlTypes.FilteredLogEvent
	NextToken *string
}

// NewClient creates a new CloudWatch Logs client using the default AWS configuration.
// It loads AWS credentials and region from the environment or AWS config files.
func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	return &Client{
		cwl: cwl.NewFromConfig(cfg),
	}, nil
}

// ValidateLogGroupNamePattern reports whether pattern is accepted as
// DescribeLogGroups' LogGroupNamePattern.
func ValidateLogGroupNamePattern(pattern string) error {
	if len(pattern) > maxLogGroupNamePatternLength {
		return fmt.Errorf("search word is longer than %d characters", maxLogGroupNamePatternLength)
	}
	if !logGroupNamePatternRegexp.MatchString(pattern) {
		return errors.New("search word may contain only A-Z a-z 0-9 . - _ / #")
	}
	return nil
}

// FormatMessage returns an event message as exactly one line terminated by "\n",
// whether or not the original message ends with a line break.
func FormatMessage(event cwlTypes.FilteredLogEvent) string {
	return strings.TrimRight(aws.ToString(event.Message), "\r\n") + "\n"
}

// GetLogGroups retrieves one page of log groups with optional filtering.
// It supports pagination through the NextToken parameter.
func (c *Client) GetLogGroups(input *LogGroupInput) (*LogGroupOutput, error) {
	if err := ValidateLogGroupNamePattern(input.FilterPattern); err != nil {
		return nil, err
	}
	params := &cwl.DescribeLogGroupsInput{
		Limit:     aws.Int32(MaxItemsInLayout),
		NextToken: input.NextToken,
	}
	if input.FilterPattern != "" {
		params.LogGroupNamePattern = aws.String(input.FilterPattern)
	}

	for {
		res, err := c.cwl.DescribeLogGroups(input.Ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to describe log groups: %w", err)
		}
		// With LogGroupNamePattern a page can be empty while more pages
		// follow; skip such pages so the UI never shows an empty page with a
		// "next" link.
		if len(res.LogGroups) > 0 || !hasNewToken(params.NextToken, res.NextToken) {
			return &LogGroupOutput{
				LogGroups: res.LogGroups,
				NextToken: res.NextToken,
			}, nil
		}
		params.NextToken = res.NextToken
	}
}

// GetLogStreams retrieves one page of log streams for a log group.
// Results are ordered by last event time in descending order and support pagination.
func (c *Client) GetLogStreams(input *LogStreamInput) (*LogStreamOutput, error) {
	params := &cwl.DescribeLogStreamsInput{
		LogGroupName: aws.String(input.LogGroupName),
		Limit:        aws.Int32(MaxItemsInLayout),
		OrderBy:      cwlTypes.OrderByLastEventTime,
		Descending:   aws.Bool(true),
		NextToken:    input.NextToken,
	}

	for {
		res, err := c.cwl.DescribeLogStreams(input.Ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to describe log streams: %w", err)
		}
		if len(res.LogStreams) > 0 || !hasNewToken(params.NextToken, res.NextToken) {
			return &LogStreamOutput{
				LogStreams: res.LogStreams,
				NextToken:  res.NextToken,
			}, nil
		}
		params.NextToken = res.NextToken
	}
}

// GetLogEvents retrieves up to MaxLogEvents log events within the time range.
// It can filter by multiple log streams and supports pattern-based filtering.
func (c *Client) GetLogEvents(input *LogEventInput) (*LogEventOutput, error) {
	params := filterLogEventsParams(input)
	out := &LogEventOutput{}

	// A FilterLogEvents page can be partially filled or empty while more
	// events exist, so keep paging until the limit is reached.
	for {
		params.Limit = aws.Int32(int32(MaxLogEvents - len(out.LogEvents)))
		res, err := c.cwl.FilterLogEvents(input.Ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to filter log events: %w", err)
		}
		out.LogEvents = append(out.LogEvents, res.Events...)
		if !hasNewToken(params.NextToken, res.NextToken) {
			return out, nil
		}
		if len(out.LogEvents) >= MaxLogEvents {
			out.NextToken = res.NextToken
			return out, nil
		}
		params.NextToken = res.NextToken
	}
}

// WriteLogEvents fetches all log events matching the input and writes them to
// input.OutputFile (DefaultOutputFile when empty), one event per line.
// It returns the absolute path written and the number of events.
// The file is replaced only when every event was written; on error an
// existing file is left untouched.
func (c *Client) WriteLogEvents(input *LogEventInput) (path string, count int, err error) {
	outputFile := input.OutputFile
	if outputFile == "" {
		outputFile = DefaultOutputFile
	}
	path, err = filepath.Abs(outputFile)
	if err != nil {
		return "", 0, fmt.Errorf("failed to resolve output file: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return path, 0, fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()

	w := bufio.NewWriter(tmp)
	paginator := cwl.NewFilterLogEventsPaginator(c.cwl, filterLogEventsParams(input), func(o *cwl.FilterLogEventsPaginatorOptions) {
		o.Limit = 10000
	})
	for paginator.HasMorePages() {
		res, err := paginator.NextPage(input.Ctx)
		if err != nil {
			return path, count, fmt.Errorf("unable to get log events: %w", err)
		}
		for _, event := range res.Events {
			if _, err := w.WriteString(FormatMessage(event)); err != nil {
				return path, count, fmt.Errorf("failed to write log message: %w", err)
			}
			count++
		}
	}

	if err = w.Flush(); err != nil {
		return path, count, fmt.Errorf("failed to write log message: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return path, count, fmt.Errorf("failed to close output file: %w", err)
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return path, count, fmt.Errorf("failed to replace output file: %w", err)
	}
	return path, count, nil
}

func filterLogEventsParams(input *LogEventInput) *cwl.FilterLogEventsInput {
	params := &cwl.FilterLogEventsInput{
		LogGroupName: aws.String(input.LogGroupName),
		StartTime:    aws.Int64(input.StartTime.UnixMilli()),
		EndTime:      aws.Int64(input.EndTime.UnixMilli()),
	}
	if len(input.LogStreamNames) > 0 {
		params.LogStreamNames = input.LogStreamNames
	}
	if input.FilterPattern != "" {
		params.FilterPattern = aws.String(input.FilterPattern)
	}
	return params
}

// hasNewToken reports whether next continues a listing that was requested
// with prev. The SDK paginators also stop on a repeated token, which some
// services return at the end of a listing.
func hasNewToken(prev, next *string) bool {
	return next != nil && *next != "" && (prev == nil || *next != *prev)
}
