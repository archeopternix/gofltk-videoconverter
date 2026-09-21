package virtualdub

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/archeopternix/gofltk-videoconverter/medialang/config"
)

type Runner struct {
	Tool   config.VirtualDubTool
	Pather *PathConverter
}

func (r Runner) Run(ctx context.Context, jobsFile string) ([]byte, error) {
	windowsJobsFile, err := r.Pather.ToWindowsPath(jobsFile)
	if err != nil {
		return nil, fmt.Errorf("convert VirtualDub jobs path: %w", err)
	}
	args := make([]string, len(r.Tool.Arguments))
	for i, argument := range r.Tool.Arguments {
		argument = strings.ReplaceAll(argument, "{{.JobsFile}}", jobsFile)
		argument = strings.ReplaceAll(argument, "{{.JobsFileWindows}}", windowsJobsFile)
		args[i] = argument
	}

	cmd := exec.CommandContext(ctx, r.Tool.Executable, args...)
	cmd.Env = os.Environ()
	for key, value := range r.Tool.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("VirtualDub failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}
