package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type awsResource struct {
	logGroups  []cwlTypes.LogGroup
	logStreams []cwlTypes.LogStream
	client     *cwl.Client
}

func (a *awsResource) getLogEvents(logGroupName string) {
	input := &cwl.FilterLogEventsInput{
		LogGroupName: aws.String(logGroupName),
	}
	paginator := cwl.NewFilterLogEventsPaginator(a.client, input, func(o *cwl.FilterLogEventsPaginatorOptions) {
		o.Limit = 10000
	})

	// for paginator.HasMorePages() {
	paginator.HasMorePages()
	res, err := paginator.NextPage(context.TODO())
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}

	// write the log event to the file
	for _, event := range res.Events {
		log.Println(aws.ToString(event.Message))
	}

	// a.logGroups = append(a.logGroups, res.LogStreams...)
	// }
}

func (a *awsResource) getLogGroups() {
	input := &cwl.DescribeLogGroupsInput{}
	paginator := cwl.NewDescribeLogGroupsPaginator(a.client, input, func(o *cwl.DescribeLogGroupsPaginatorOptions) {
		o.Limit = 50
	})

	for paginator.HasMorePages() {

		res, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Fatalf("unable to list tables, %v", err)
		}

		a.logGroups = append(a.logGroups, res.LogGroups...)
	}
}
