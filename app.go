package main

import (
	"context"
	"log"
	"strconv"
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
			tvApp:   tview.NewApplication(),
			widgets: make(map[Widget]tview.Primitive),
			layouts: make(map[Layout]tview.Primitive),
			lEForm: &logEventForm{
				enableOutputFile: false,
			},
			logGroup: &logGroup{
				direction:  Home,
				pageTokens: make(map[int]*string),
			},
			logStream: &logStream{
				direction:  Home,
				pageTokens: make(map[int]*string),
			},
		},
		awsr: &awsResource{
			client: cwl.NewFromConfig(cfg),
		},
	}
}

func (a *App) Run() {
	if err := a.gui.tvApp.SetRoot(a.gui.pages, true).
		EnableMouse(true).
		SetFocus(a.gui.widgets[LogGroupTable]).
		Run(); err != nil {
		panic(err)
	}
}

func getDaysByMonth(month string) []string {
	var days []string
	y := 2024
	intMonth, err := strconv.Atoi(month)
	if err != nil {
		log.Fatalf("unable to convert month to integer, %v", err)
	}
	m := time.Month(intMonth)

	// Start from the first day of the month
	startDate := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	// Get the number of days in the month
	for d := startDate; d.Month() == m; d = d.AddDate(0, 0, 1) {
		// Convert day to a string without leading zero
		dayStr := strconv.Itoa(d.Day())
		days = append(days, dayStr)
	}
	return days
}
