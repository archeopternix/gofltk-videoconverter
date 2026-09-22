package avisynth

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	embedded "github.com/archeopternix/gofltk-videoconverter/config"
	"github.com/archeopternix/gofltk-videoconverter/medialang/filter"
	"github.com/archeopternix/gofltk-videoconverter/medialang/media"
)

type WindowsPather interface {
	ToWindowsPath(path string) (string, error)
}

type Compiler struct {
	AviSynthPath string
	Pather       WindowsPather
}

func (c Compiler) Compile(input media.Artifact, profile *filter.AviSynthProfile, outputPath string) (media.Artifact, error) {
	filename := "avisynth/profiles/" + filepath.Base(profile.Config.Profile) + ".avs.tpl"
	tpl, err := template.New("Avisynth").Option("missingkey=error").ParseFS(embedded.Files, filename)
	if err != nil {
		return media.Artifact{}, fmt.Errorf("parse AviSynth profile %q: %w", profile.Config.Profile, err)
	}

	inFile, err := c.Pather.ToWindowsPath(input.Path)
	if err != nil {
		return media.Artifact{}, fmt.Errorf("convert AviSynth input path %q: %w", input.Path, err)
	}
	avisynthPath, err := optionalWindowsPath(c.Pather, c.AviSynthPath)
	if err != nil {
		return media.Artifact{}, fmt.Errorf("convert AviSynth path: %w", err)
	}

	values := make(map[string]any, len(profile.Config.Values)+14)
	for key, value := range profile.Config.Values {
		values[key] = value
	}
	values["AvisynthPath"] = avisynthPath
	values["InFile"] = inFile
	values["Deinterlace"] = input.Media.ScanType == media.ScanInterlaced
	values["ConvertYV"] = !input.Media.IsYV12
	values["Preset"] = profile.Config.Preset
	values["ResizeX"] = profile.Config.ResizeX
	values["ResizeY"] = profile.Config.ResizeY
	values["Width"] = input.Media.Width
	values["Height"] = input.Media.Height
	values["FPS"] = input.Media.FPS
	values["FieldOrder"] = input.Media.FieldOrder
	values["PixelFormat"] = input.Media.PixelFormat
	values["ColorSpace"] = input.Media.ColorSpace
	values["ColorFamily"] = input.Media.ColorFamily

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return media.Artifact{}, fmt.Errorf("create AviSynth directory: %w", err)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return media.Artifact{}, fmt.Errorf("create AviSynth script %q: %w", outputPath, err)
	}
	executeErr := tpl.ExecuteTemplate(file, "Avisynth", values)
	closeErr := file.Close()
	if executeErr != nil {
		return media.Artifact{}, fmt.Errorf("render AviSynth profile %q: %w", profile.Config.Profile, executeErr)
	}
	if closeErr != nil {
		return media.Artifact{}, fmt.Errorf("close AviSynth script %q: %w", outputPath, closeErr)
	}
	return media.Artifact{Path: outputPath, Type: media.ArtifactAviSynth, Media: input.Media}, nil
}

func optionalWindowsPath(pather WindowsPather, path string) (string, error) {
	if path == "" {
		return "", nil
	}
	return pather.ToWindowsPath(path)
}
