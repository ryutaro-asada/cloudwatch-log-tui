// Package main provides the entry point for the CloudWatch Log TUI application.
// This terminal-based user interface allows users to browse and search AWS CloudWatch logs interactively.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/app"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/config"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/state"
)

// main initializes the application configuration, sets up AWS client connections,
// and launches the terminal user interface for browsing CloudWatch logs.
func main() {
	if err := run(); err != nil {
		log.Print(err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var opts aws.Options
	flag.StringVar(&opts.Profile, "profile", "", "AWS shared config profile (default: AWS_PROFILE, or the default profile)")
	flag.StringVar(&opts.Region, "region", "", "AWS region (default: AWS_REGION, or the region of the profile)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags]\n\nBrowse AWS CloudWatch Logs in the terminal.\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	cfg := config.New()

	logFile, err := cfg.InitLogging()
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	awsClient, err := aws.NewClient(ctx, opts)
	if err != nil {
		return fmt.Errorf("error initializing AWS client: %w", err)
	}

	tui := app.New(ctx, awsClient)

	// tview reads Ctrl-C as a key event, so a signal only arrives from outside
	// (kill, terminal hangup). Stop the UI so the terminal is restored.
	go func() {
		<-ctx.Done()
		tui.Stop()
	}()

	tui.LoadAccount()
	tui.LoadLogGroups(state.Home)

	err = tui.Run()
	stop()
	if err != nil {
		return fmt.Errorf("error running application: %w", err)
	}
	return nil
}
