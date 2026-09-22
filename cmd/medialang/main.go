package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/archeopternix/gofltk-videoconverter/medialang"
	"github.com/archeopternix/gofltk-videoconverter/medialang/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	runID := time.Now().Format("20060102T150405.000")

	defaultConfigFile, err := configFileForOS(runtime.GOOS)
	if err != nil {
		slog.Error("configuration failed", "stage", "preparation", "run_id", runID, "error", err)
		os.Exit(1)
	}

	flag.Usage = func() {
		output := flag.CommandLine.Output()
		fmt.Fprintf(output, "Usage: %s [options] <file-or-pattern> [<file-or-pattern> ...]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintln(output, "Inputs may be individual files or quoted patterns such as *.mov or C:\\Videos\\*.mp4.")
		fmt.Fprintln(output, "\nOptions:")
		flag.PrintDefaults()
		fmt.Fprintln(output, "\nExamples:")
		fmt.Fprintf(output, "  %s video.mov\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(output, "  %s video1.mov video2.mp4\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(output, "  %s \"*.mov\"\n", filepath.Base(os.Args[0]))
	}
	configFile := flag.String("config", defaultConfigFile, "path to the MediaLang app config")
	help := flag.Bool("help", false, "show usage")
	shortHelp := flag.Bool("h", false, "show usage")
	questionHelp := flag.Bool("?", false, "show usage")
	flag.Parse()
	if *help || *shortHelp || *questionHelp {
		flag.Usage()
		return
	}
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	files, err := expandInputs(flag.Args())
	if err != nil {
		slog.Error("input expansion failed", "stage", "preparation", "run_id", runID, "error", err)
		os.Exit(2)
	}

	app, err := config.Load(*configFile)
	if err != nil {
		slog.Error("configuration failed", "stage", "preparation", "run_id", runID, "error", err)
		os.Exit(1)
	}
	runner := medialang.NewRunner(app, files, runID)
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
		slog.Error("run failed", "stage", "finished", "run_id", runID, "error", runErr)
		os.Exit(1)
	}
}

func expandInputs(inputs []string) ([]string, error) {
	var files []string
	for _, input := range inputs {
		if !strings.ContainsAny(input, "*?[") {
			files = append(files, input)
			continue
		}
		matches, err := filepath.Glob(input)
		if err != nil {
			return nil, fmt.Errorf("invalid file pattern %q: %w", input, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("file pattern %q matched no files", input)
		}
		files = append(files, matches...)
	}
	return files, nil
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
