package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"time"

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

func (a *awsResource) getLogEvents(input logEventInut) {
	// input := &cwl.FilterLogEventsInput{
	// 	LogGroupName:  aws.String(lef.logGroupName),
	// 	StartTime:     aws.Int64(startTime(lef)),
	// 	EndTime:       aws.Int64(endTime(lef)),
	// 	FilterPattern: aws.String(lef.filterPatern),
	// }

	paginator := cwl.NewFilterLogEventsPaginator(a.client, input.awsInput, func(o *cwl.FilterLogEventsPaginatorOptions) {
		o.Limit = 10000
	})

	var outputFile string
	if input.outputFile != "" {
		outputFile = input.outputFile
	} else {
		outputFile = "app.log"
	}
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("unable to create file, %v", err)
	}
	defer f.Close()
	bf := bufio.NewWriter(f)
	defer bf.Flush()

	for paginator.HasMorePages() {
		res, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Fatalf("unable to get next page, %v", err)
		}

		for _, event := range res.Events {
			_, err = bf.WriteString(aws.ToString(event.Message) + "\n")
			if err != nil {
				log.Fatalf("unable to write to file, %v", err)
			}
		}
	}
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
		log.Printf("filterPatern: %s", filterPatern)
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
	log.Printf("currentPageLogGroup: %d", a.currentPageLogGroup)

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
		log.Printf("prefixPatern: %s", prefixPatern)
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
	log.Printf("currentPageLogGroup: %d", a.currentPageLogStream)

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

func startTime(lef *logEventForm) *int64 {
	if lef.startTimeSelected {
		return aws.Int64(time.Date(lef.startYear, lef.startMonth, lef.startDay, lef.startHour, lef.startMinute, 0, 0, time.Local).UnixMilli())
	}
	return nil
}

func endTime(lef *logEventForm) *int64 {
	if lef.endTimeSelected {
		return aws.Int64(time.Date(lef.endYear, lef.endMonth, lef.endDay, lef.endHour, lef.endMinute, 0, 0, time.Local).UnixMilli())
	}
	return nil
}

func filterPattern(lef *logEventForm) *string {
	if lef.enableFilterPatern {
		return aws.String(lef.filterPatern)
	}
	return nil
}
