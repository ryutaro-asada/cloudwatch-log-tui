package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/config"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/state"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/ui"
)

func main() {
	// Initialize configuration
	cfg := config.New()

	// Setup logging
	logFile, err := cfg.InitLogging()
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx)
	if err != nil {
		log.Fatalf("error initializing AWS client: %v", err)
	}

	// Create UI
	app := ui.New(ctx, awsClient)

	go app.LoadLogGroups(state.Home)

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatalf("error running application: %v", err)
	}
}
