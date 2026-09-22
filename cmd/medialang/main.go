package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/archeopternix/gofltk-videoconverter/medialang"
	"github.com/archeopternix/gofltk-videoconverter/medialang/config"
)

var files []string

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	defaultConfigFile, err := configFileForOS(runtime.GOOS)
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	configFile := flag.String("config", defaultConfigFile, "path to the MediaLang app config")

	//	files = []string{"/home/archeopternix/Videos/IMGA0291.MP4"}

	files = []string{`C:\Users\Andreas Eisner\Videos\2026 Yasmin Geburtstag\20260228_200526.mp4`}

	flag.Parse()
	/*	if flag.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "usage: medialang [-config config/app.yaml] <video> [video...]")
			os.Exit(2)
		}
	*/
	if flag.NArg() > 0 {
		files = flag.Args()
	}

	app, err := config.Load(*configFile)
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	runner := medialang.NewRunner(app, files)
	result, runErr := runner.Run(context.Background())
	if result != nil {
		for _, file := range result.Files {
			if file.Error != nil {
				fmt.Printf("%-10s %s [stage=%s]: %v\n", file.Status, file.Input, file.Stage, file.Error)
			} else {
				fmt.Printf("%-10s %s -> %s [stage=%s]\n", file.Status, file.Input, file.Output, file.Stage)
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

func configFileForOS(goos string) (string, error) {
	switch goos {
	case "windows":
		return "config/app.windows.yaml", nil
	case "linux":
		return "config/app.linux.yaml", nil
	default:
		return "", fmt.Errorf("unsupported operating system %q; only windows and linux are supported", goos)
	}
}
