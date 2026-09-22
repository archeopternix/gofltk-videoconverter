package virtualdub

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/archeopternix/gofltk-videoconverter/medialang/config"
)

type Runner struct {
	Tool   config.VirtualDubTool
	Pather *PathConverter
	Logger *slog.Logger
}

type Result struct {
	Output   string
	ExitCode int
}

func (r Runner) Run(ctx context.Context, jobsFile string) (Result, error) {
	logger := r.Logger
	if logger == nil {
		logger = slog.Default()
	}

	jobsFile, err := filepath.Abs(jobsFile)
	if err != nil {
		return Result{ExitCode: -1}, fmt.Errorf("resolve VirtualDub jobs path: %w", err)
	}

	cmd, err := r.command(ctx, jobsFile, logger)
	if err != nil {
		return Result{ExitCode: -1}, err
	}
	cmd.Env = os.Environ()
	for key, value := range r.Tool.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	output, err := cmd.CombinedOutput()
	result := Result{Output: strings.TrimSpace(string(output)), ExitCode: 0}
	if err != nil {
		result.ExitCode = -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		return result, fmt.Errorf("execute VirtualDub: %w", err)
	}
	return result, nil
}

func (r Runner) command(ctx context.Context, jobsFile string, logger *slog.Logger) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "windows":
		batchFile, err := filepath.Abs(filepath.Join("cmd", "medialang", "vdub.bat"))
		if err != nil {
			return nil, fmt.Errorf("resolve VirtualDub batch path: %w", err)
		}
		info, err := os.Stat(batchFile)
		if err != nil {
			return nil, fmt.Errorf("access VirtualDub batch file %q: %w", batchFile, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("VirtualDub batch path is not a regular file: %s", batchFile)
		}

		commandProcessor := strings.TrimSpace(os.Getenv("ComSpec"))
		if commandProcessor == "" {
			commandProcessor = "cmd.exe"
		}
		args := []string{"/D", "/C", "call", batchFile, r.Tool.Executable, jobsFile, "/x"}
		cmd := exec.CommandContext(ctx, commandProcessor, args...)
		logger.Debug("external command prepared",
			"stage", "virtualdub",
			"command_processor", commandProcessor,
			"batch_file", batchFile,
			"virtualdub", r.Tool.Executable,
			"jobs_file", jobsFile,
			"exit_argument", "/x",
			"command", cmd.String(),
		)
		return cmd, nil

	case "linux":
		args := make([]string, len(r.Tool.Arguments))
		for i, argument := range r.Tool.Arguments {
			args[i] = strings.ReplaceAll(argument, "{{.JobsFile}}", jobsFile)
		}
		cmd := exec.CommandContext(ctx, r.Tool.Executable, args...)
		logger.Debug("external command prepared",
			"stage", "virtualdub",
			"executable", r.Tool.Executable,
			"jobs_file", jobsFile,
			"args", args,
		)
		return cmd, nil

	default:
		return nil, fmt.Errorf("unsupported operating system %q for VirtualDub; only windows and linux are supported", runtime.GOOS)
	}
}
