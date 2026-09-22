package ffmpeg

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"

	"github.com/archeopternix/gofltk-videoconverter/medialang/filter"
)

type Runner struct {
	Executable string
	Logger     *slog.Logger
}

type Result struct {
	Output   string
	ExitCode int
}

func (r Runner) Run(ctx context.Context, input, output string, config filter.ZScaleConfig) (Result, error) {
	filterChain := fmt.Sprintf("zscale=w=%d:h=%d:filter=%s,format=%s", config.Width, config.Height, config.Filter, config.PixelFormat)
	if config.PadWidth > 0 && config.PadHeight > 0 {
		filterChain += fmt.Sprintf(",pad=%d:%d:%d:%d:black,setsar=1", config.PadWidth, config.PadHeight, config.PadX, config.PadY)
	}
	args := []string{"-y", "-i", input, "-vf", filterChain, "-c:v", config.VideoCodec}
	if config.CRF != nil {
		args = append(args, "-crf", strconv.Itoa(*config.CRF))
	}
	if config.Preset != "" {
		args = append(args, "-preset", config.Preset)
	}
	args = append(args, "-c:a", config.AudioCodec)
	if config.AudioBitrate != "" {
		args = append(args, "-b:a", config.AudioBitrate)
	}
	args = append(args, config.ExtraArgs...)
	args = append(args, output)

	executable := strings.TrimSpace(r.Executable)
	if executable == "" {
		return Result{ExitCode: -1}, fmt.Errorf("config key tools.ffmpeg.path is required for ffmpeg.zscale")
	}
	command := exec.CommandContext(ctx, executable, args...)
	logger := r.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Debug("external command prepared", "stage", "scaling", "path", command.Path, "args", command.Args)
	combined, err := command.CombinedOutput()
	result := Result{Output: strings.TrimSpace(string(combined)), ExitCode: 0}
	if err != nil {
		result.ExitCode = -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		return result, fmt.Errorf("execute FFmpeg: %w", err)
	}
	return result, nil
}
