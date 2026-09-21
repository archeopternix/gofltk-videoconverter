package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/archeopternix/gofltk-videoconverter/medialang"
	"github.com/archeopternix/gofltk-videoconverter/medialang/config"
)

func main() {
	configFile := flag.String("config", "config/app.yaml", "path to the MediaLang app config")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: medialang [-config config/app.yaml] <video> [video...]")
		os.Exit(2)
	}

	app, err := config.Load(*configFile)
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	runner := medialang.NewRunner(app, flag.Args())
	result, runErr := runner.Run(context.Background())
	if result != nil {
		for _, file := range result.Files {
			if file.Error != nil {
				fmt.Printf("%-10s %s: %v\n", file.Status, file.Input, file.Error)
			} else {
				fmt.Printf("%-10s %s -> %s\n", file.Status, file.Input, file.Output)
			}
		}
		if result.JobsFile != "" {
			fmt.Printf("jobs: %s\n", result.JobsFile)
		}
	}
	if runErr != nil {
		slog.Error("run failed", "error", runErr)
		os.Exit(1)
	}
}
