package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/archeopternix/gofltk-videoconverter/medialang/media"
)

type FFProbe struct {
	Executable string
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeFormat struct {
	FormatName string `json:"format_name"`
}

type ffprobeStream struct {
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AvgFrameRate string `json:"avg_frame_rate"`
	RFrameRate   string `json:"r_frame_rate"`
	FieldOrder   string `json:"field_order"`
	PixelFormat  string `json:"pix_fmt"`
	ColorSpace   string `json:"color_space"`
}

func (p FFProbe) Probe(ctx context.Context, filename string) (media.MediaSpec, error) {
	executable := p.Executable
	if executable == "" {
		executable = "ffprobe"
	}
	cmd := exec.CommandContext(ctx, executable,
		"-v", "error",
		"-show_streams",
		"-show_format",
		"-of", "json",
		filename,
	)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return media.MediaSpec{}, fmt.Errorf("ffprobe %q: %w: %s", filename, err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return media.MediaSpec{}, fmt.Errorf("ffprobe %q: %w", filename, err)
	}

	var result ffprobeOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return media.MediaSpec{}, fmt.Errorf("decode ffprobe output for %q: %w", filename, err)
	}

	var video *ffprobeStream
	var audio *ffprobeStream
	for i := range result.Streams {
		stream := &result.Streams[i]
		switch stream.CodecType {
		case "video":
			if video == nil {
				video = stream
			}
		case "audio":
			if audio == nil {
				audio = stream
			}
		}
	}
	if video == nil {
		return media.MediaSpec{}, fmt.Errorf("no video stream in %q", filename)
	}

	fps := rational(video.AvgFrameRate)
	if fps == 0 {
		fps = rational(video.RFrameRate)
	}
	scan, order := scanType(video.FieldOrder)
	spec := media.MediaSpec{
		Container:   normalize(result.Format.FormatName),
		Codec:       normalize(video.CodecName),
		Width:       video.Width,
		Height:      video.Height,
		FPS:         fps,
		ScanType:    scan,
		FieldOrder:  order,
		PixelFormat: normalize(video.PixelFormat),
		ColorSpace:  normalize(video.ColorSpace),
		ColorFamily: colorFamily(video.PixelFormat),
	}
	if audio != nil {
		spec.AudioCodec = normalize(audio.CodecName)
	}
	return spec, nil
}

func rational(value string) float64 {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		result, _ := strconv.ParseFloat(value, 64)
		return result
	}
	numerator, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	denominator, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func scanType(value string) (media.ScanType, media.FieldOrder) {
	switch normalize(value) {
	case "progressive":
		return media.ScanProgressive, media.FieldOrderUnknown
	case "tt", "tb":
		return media.ScanInterlaced, media.FieldOrderTFF
	case "bb", "bt":
		return media.ScanInterlaced, media.FieldOrderBFF
	default:
		return media.ScanUnknown, media.FieldOrderUnknown
	}
}

func colorFamily(pixelFormat string) string {
	value := normalize(pixelFormat)
	switch {
	case strings.HasPrefix(value, "rgb"), strings.HasPrefix(value, "bgr"), strings.HasPrefix(value, "gbr"):
		return "rgb"
	case strings.HasPrefix(value, "yuv"), strings.HasPrefix(value, "yuva"), strings.HasPrefix(value, "nv"):
		return "yuv"
	case strings.HasPrefix(value, "gray"):
		return "gray"
	default:
		return value
	}
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
