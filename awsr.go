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
	pageTokenLogGroup   map[int]*string
	hasNextLogGroup     bool
	hasPrevLogGroup     bool

	// pagesLogGroup  *cwl.DescribeLogGroupsPaginator
	logStreams           []cwlTypes.LogStream
	currentPageLogStream int
	nextTokensLogStream  []*string
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

// func (a *awsResource) getPageLogGroups() {
// 	input := &cwl.DescribeLogGroupsInput{}
// 	a.pagesLogGroup = cwl.NewDescribeLogGroupsPaginator(a.client, input, func(o *cwl.DescribeLogGroupsPaginatorOptions) {
// 		o.Limit = 50
// 	})
// }

func (a *awsResource) getLogGroups(direct Direction) {
	// input := &cwl.DescribeLogGroupsInput{}
	// paginator := cwl.NewDescribeLogGroupsPaginator(a.client, input, func(o *cwl.DescribeLogGroupsPaginatorOptions) {
	// 	o.Limit = 50
	// })

	params := &cwl.DescribeLogGroupsInput{
		Limit: aws.Int32(maxItemsInPage),
	}
	if direct == Next && a.hasNextLogGroup {
		params.NextToken = a.pageTokenLogGroup[a.currentPageLogGroup+1]
	} else if direct == Prev && a.hasPrevLogGroup {
		params.NextToken = a.pageTokenLogGroup[a.currentPageLogGroup-1]
	}

	res, err := a.client.DescribeLogGroups(context.TODO(), params)
	// log.Println("next token ...................")
	// log.Println(res.NextToken)
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}
	a.logGroups = res.LogGroups

	if direct == Next {
		a.currentPageLogGroup++
	} else if direct == Prev {
		a.currentPageLogGroup--
	}

	if res.NextToken != nil {
		a.pageTokenLogGroup[a.currentPageLogGroup+1] = res.NextToken
		a.hasNextLogGroup = true
	} else {
		a.hasNextLogGroup = false
	}

	if a.currentPageLogGroup > 1 {
		a.hasPrevLogGroup = true
	} else {
		a.hasPrevLogGroup = false
	}

	// log.Println("map ...................", a.pageTokenLogGroup)
	// log.Println("currentPage ...................", a.currentPageLogGroup)

	// for paginator.HasMorePages() {
	//
	// 	res, err := paginator.NextPage(context.TODO())
	// 	if err != nil {
	// 		log.Fatalf("unable to list tables, %v", err)
	// 	}
	//
	// 	a.logGroups = append(a.logGroups, res.LogGroups...)
	// }
}

// func (a *awsResource) getLogStreams(logGroupName string) {
// 	input := &cwl.DescribeLogStreamsInput{
// 		LogGroupName: aws.String(logGroupName),
// 	}
// 	a.pagesLogStream = cwl.NewDescribeLogStreamsPaginator(a.client, input, func(o *cwl.DescribeLogStreamsPaginatorOptions) {
// 		o.Limit = 50
// 	})
// }

func (a *awsResource) getLogStreams(logGroupName string) {
	input := &cwl.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroupName),
	}
	paginator := cwl.NewDescribeLogStreamsPaginator(a.client, input, func(o *cwl.DescribeLogStreamsPaginatorOptions) {
		o.Limit = 50
	})

	for paginator.HasMorePages() {

		res, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Fatalf("unable to list tables, %v", err)
		}

		a.logStreams = append(a.logStreams, res.LogStreams...)
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
