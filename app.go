package main

import (
	"context"
	"log"
	"time"

	"github.com/rivo/tview"

	"github.com/aws/aws-sdk-go-v2/config"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type App struct {
	gui  *gui
	awsr *awsResource
}

func NewApp() *App {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	return &App{
		gui: &gui{
			tvApp: tview.NewApplication(),
			widgets: make(map[string]tview.Primitive),
			layouts: make(map[string]*tview.Flex),
			lEFrom: &logEventForm{},
		},
		awsr: &awsResource{
			client: cwl.NewFromConfig(cfg),
		},
	}
}

func (a *App) Run() {
	if err := a.gui.tvApp.SetRoot(a.gui.pages, true).
		EnableMouse(true).
		SetFocus(a.gui.widgets[logGroupList]).
		Run(); err != nil {
		panic(err)
	}
}

func getDaysByMonth(year int, month time.Month) []time.Time {
	var days []time.Time
	// Start from the first day of the month
	startDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	// Get the number of days in the month
	for d := startDate; d.Month() == month; d = d.AddDate(0, 0, 1) {
		days = append(days, d)
	}
	return days
}
