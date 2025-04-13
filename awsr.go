package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

const maxItemsInPage int32 = 50

type awsResource struct {
	logGroups []cwlTypes.LogGroup

	logStreams []cwlTypes.LogStream

	client *cwl.Client
}

func (a *awsResource) getLogEvents(input logEventInput) (*cwl.FilterLogEventsOutput, error) {
	res, err := a.client.FilterLogEvents(context.TODO(), input.awsInput)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *awsResource) getLogGroups(lg *logGroup) {
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
		params.NextToken = lg.pageTokens[lg.currentPage+1]

	case Prev:
		// if currentPageLogGroup == 1, a.pageTokenLogGroup will return nil and params.NextToken will be nil
		// so, first page will be fetched
		params.NextToken = lg.pageTokens[lg.currentPage-1]

	case Home:
		lg.pageTokens = make(map[int]*string)

	}

	res, err := a.client.DescribeLogGroups(context.TODO(), params)
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}
	a.logGroups = res.LogGroups

	switch direct {
	case Next:
		lg.currentPage++
	case Prev:
		lg.currentPage--
	case Home:
		lg.currentPage = 1
	}

	if res.NextToken != nil && len(res.LogGroups) == int(maxItemsInPage) {
		log.Printf("next token: %s", *res.NextToken)
		lg.pageTokens[lg.currentPage+1] = res.NextToken
		lg.hasNext = true
	} else {
		lg.hasNext = false
	}

	if lg.currentPage > 1 {
		lg.hasPrev = true
	} else {
		lg.hasPrev = false
	}
}

func (a *awsResource) getLogStreams(ls *logStream) {
	prefixPatern := ls.prefixPatern
	direct := ls.direction
	logGroupName := ls.logGroupName

	params := &cwl.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroupName),
		Limit:        aws.Int32(maxItemsInPage),
	}

	if prefixPatern != "" {
		params.LogStreamNamePrefix = aws.String(prefixPatern)
	}

	switch direct {
	case Next:
		params.NextToken = ls.pageTokens[ls.currentPage+1]
	case Prev:
		// if currentPageLogGroup == 1, a.pageTokenLogGroup will return nil and params.NextToken will be nil
		// so, first page will be fetched
		params.NextToken = ls.pageTokens[ls.currentPage-1]
	case Home:
		ls.pageTokens = make(map[int]*string)
	}

	res, err := a.client.DescribeLogStreams(context.TODO(), params)
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}
	a.logStreams = res.LogStreams

	switch direct {
	case Next:
		ls.currentPage++
	case Prev:
		ls.currentPage--
	case Home:
		ls.currentPage = 1
	}

	// TODO: add test
	if res.NextToken != nil && len(res.LogStreams) == int(maxItemsInPage) {
		ls.pageTokens[ls.currentPage+1] = res.NextToken
		ls.hasNext = true
	} else {
		ls.hasNext = false
	}

	// TODO: add test
	if ls.currentPage > 1 {
		ls.hasPrev = true
	} else {
		ls.hasPrev = false
	}
}
