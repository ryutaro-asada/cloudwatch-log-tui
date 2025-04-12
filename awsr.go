package main

import (
	// "bufio"
	"context"
	"log"
	// "os"
	// "time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

const maxItemsInPage int32 = 50

type awsResource struct {
	logGroups           []cwlTypes.LogGroup
	currentPageLogGroup int
	pageTokensLogGroup  map[int]*string
	hasNextLogGroup     bool
	hasPrevLogGroup     bool

	// pagesLogGroup  *cwl.DescribeLogGroupsPaginator
	logStreams           []cwlTypes.LogStream
	currentPageLogStream int
	pageTokensLogStream  map[int]*string
	hasNextLogStream     bool
	hasPrevLogStream     bool

	// pagesLogStream *cwl.DescribeLogStreamsPaginator
	client *cwl.Client
}

func (a *awsResource) getLogEvents(input logEventInput) (*cwl.FilterLogEventsOutput, error) {
	res, err := a.client.FilterLogEvents(context.TODO(), input.awsInput)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *awsResource) getLogGroups(lg logGroup) {
	// if direction is next,
	// - increment currentPageLogGroup after successful call
	// - set hasNextLogGroup to true if NextToken of responce is not nil
	// - set hasPrevLogGroup to true if currentPageLogGroup > 1
	// - set NextToken of currentPageLogGroup+1

	// if direction is prev, decrement currentPageLogGroup

	filterPatern := lg.filterPatern
	direct := lg.direction

	params := &cwl.DescribeLogGroupsInput{
		Limit: aws.Int32(maxItemsInPage),
	}

	// TODO: add test
	if filterPatern != "" {
		params.LogGroupNamePattern = aws.String(filterPatern)
	}

	// TODO: add test
	switch direct {
	case Next:
		params.NextToken = a.pageTokensLogGroup[a.currentPageLogGroup+1]

	case Prev:
		// if currentPageLogGroup == 1, a.pageTokenLogGroup will return nil and params.NextToken will be nil
		// so, first page will be fetched
		params.NextToken = a.pageTokensLogGroup[a.currentPageLogGroup-1]

	case Home:
		a.currentPageLogGroup = 1
		a.pageTokensLogGroup = make(map[int]*string)

	}

	res, err := a.client.DescribeLogGroups(context.TODO(), params)
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}
	a.logGroups = res.LogGroups

	// TODO: add test
	switch direct {
	case Next:
		a.currentPageLogGroup++
	case Prev:
		a.currentPageLogGroup--
	}

	// TODO: add test
	if res.NextToken != nil && len(res.LogGroups) == int(maxItemsInPage) {
		a.pageTokensLogGroup[a.currentPageLogGroup+1] = res.NextToken
		a.hasNextLogGroup = true
	} else {
		a.hasNextLogGroup = false
	}

	// TODO: add test
	if a.currentPageLogGroup > 1 {
		a.hasPrevLogGroup = true
	} else {
		a.hasPrevLogGroup = false
	}
}

func (a *awsResource) getLogStreams(ls logStream) {
	// if direction is next,
	// - increment currentPageLogGroup after successful call
	// - set hasNextLogGroup to true if NextToken of responce is not nil
	// - set hasPrevLogGroup to true if currentPageLogGroup > 1
	// - set NextToken of currentPageLogGroup+1

	// if direction is prev, decrement currentPageLogGroup

	prefixPatern := ls.prefixPatern
	direct := ls.direction
	logGroupName := ls.logGroupName

	params := &cwl.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroupName),
		Limit:        aws.Int32(maxItemsInPage),
	}

	// TODO: add test
	if prefixPatern != "" {
		params.LogStreamNamePrefix = aws.String(prefixPatern)
	}

	// TODO: add test
	switch direct {
	case Next:
		params.NextToken = a.pageTokensLogStream[a.currentPageLogStream+1]
	case Prev:
		// if currentPageLogGroup == 1, a.pageTokenLogGroup will return nil and params.NextToken will be nil
		// so, first page will be fetched
		params.NextToken = a.pageTokensLogStream[a.currentPageLogStream-1]
	case Home:
		a.currentPageLogStream = 1
		a.pageTokensLogStream = make(map[int]*string)
	}

	res, err := a.client.DescribeLogStreams(context.TODO(), params)
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}
	a.logStreams = res.LogStreams

	// TODO: add test
	switch direct {
	case Next:
		a.currentPageLogStream++
	case Prev:
		a.currentPageLogStream--
	}

	// TODO: add test
	if res.NextToken != nil && len(res.LogStreams) == int(maxItemsInPage) {
		a.pageTokensLogStream[a.currentPageLogStream+1] = res.NextToken
		a.hasNextLogStream = true
	} else {
		a.hasNextLogStream = false
	}

	// TODO: add test
	if a.currentPageLogStream > 1 {
		a.hasPrevLogStream = true
	} else {
		a.hasPrevLogStream = false
	}
}
