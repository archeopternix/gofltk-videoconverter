package avisynth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/archeopternix/gofltk-videoconverter/medialang/filter"
	"github.com/archeopternix/gofltk-videoconverter/medialang/media"
)

type WindowsPather interface {
	ToWindowsPath(path string) (string, error)
}

type Compiler struct {
	ProfilesDir string
	Pather      WindowsPather
}

func (c Compiler) Compile(input media.Artifact, profile *filter.AviSynthProfile, outputPath string) (media.Artifact, error) {
	filename := filepath.Join(c.ProfilesDir, profile.Config.Profile+".avs.tpl")
	data, err := os.ReadFile(filename)
	if err != nil {
		return media.Artifact{}, fmt.Errorf("read AviSynth profile %q: %w", profile.Config.Profile, err)
	}

	windowsPath, err := c.Pather.ToWindowsPath(input.Path)
	if err != nil {
		return media.Artifact{}, fmt.Errorf("convert AviSynth input path %q: %w", input.Path, err)
	}
	script := string(data)
	script = strings.ReplaceAll(script, "{{.InputPath}}", input.Path)
	script = strings.ReplaceAll(script, "{{.InputPathWindows}}", windowsPath)
	script = strings.ReplaceAll(script, "{{.Width}}", fmt.Sprint(input.Media.Width))
	script = strings.ReplaceAll(script, "{{.Height}}", fmt.Sprint(input.Media.Height))
	script = strings.ReplaceAll(script, "{{.FPS}}", fmt.Sprint(input.Media.FPS))
	script = strings.ReplaceAll(script, "{{.FieldOrder}}", string(input.Media.FieldOrder))
	for key, value := range profile.Config.Values {
		script = strings.ReplaceAll(script, "{{."+key+"}}", value)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return media.Artifact{}, fmt.Errorf("create AviSynth directory: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(script), 0o644); err != nil {
		return media.Artifact{}, fmt.Errorf("write AviSynth script %q: %w", outputPath, err)
	}
	return media.Artifact{Path: outputPath, Type: media.ArtifactAviSynth, Media: input.Media}, nil
}
