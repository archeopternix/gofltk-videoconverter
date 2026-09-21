package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type App struct {
	Tools      Tools      `yaml:"tools"`
	Paths      Paths      `yaml:"paths"`
	Processing Processing `yaml:"processing"`
}

type Tools struct {
	FFprobe    FFprobeTool    `yaml:"ffprobe"`
	FFmpeg     FFmpegTool     `yaml:"ffmpeg"`
	VirtualDub VirtualDubTool `yaml:"virtualdub"`
}

type FFprobeTool struct {
	Path string `yaml:"path"`
}

type FFmpegTool struct {
	Path string `yaml:"path"`
}

type VirtualDubTool struct {
	Executable string            `yaml:"executable"`
	Arguments  []string          `yaml:"arguments"`
	PathMode   string            `yaml:"path_mode"`
	WineDrive  string            `yaml:"wine_drive"`
	Mappings   []WindowsPathMap  `yaml:"path_mappings"`
	Env        map[string]string `yaml:"env"`
}

type WindowsPathMap struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

type Paths struct {
	Workflows        string `yaml:"workflows"`
	AviSynthProfiles string `yaml:"avisynth_profiles"`
	AviSynth         string `yaml:"avisynth"`
	VirtualDubCodecs string `yaml:"virtualdub_codecs"`
	Deshaker         string `yaml:"deshaker"`
}

type Processing struct {
	OutputDir string `yaml:"output_dir"`
	WorkDir   string `yaml:"work_dir"`
	KeepFiles bool   `yaml:"keep_temp_files"`
}

func Load(filename string) (*App, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read app config %q: %w", filename, err)
	}

	var app App
	if err := yaml.Unmarshal(data, &app); err != nil {
		return nil, fmt.Errorf("parse app config %q: %w", filename, err)
	}

	base, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return nil, fmt.Errorf("resolve config directory: %w", err)
	}
	app.resolvePaths(base)
	if err := app.validate(); err != nil {
		return nil, err
	}
	return &app, nil
}

func (a *App) resolvePaths(base string) {
	a.Paths.Workflows = resolve(base, a.Paths.Workflows)
	a.Paths.AviSynthProfiles = resolve(base, a.Paths.AviSynthProfiles)
	a.Paths.AviSynth = resolve(base, a.Paths.AviSynth)
	a.Paths.VirtualDubCodecs = resolve(base, a.Paths.VirtualDubCodecs)
	a.Paths.Deshaker = resolve(base, a.Paths.Deshaker)
	a.Processing.OutputDir = resolve(base, a.Processing.OutputDir)
	a.Processing.WorkDir = resolve(base, a.Processing.WorkDir)
	for i := range a.Tools.VirtualDub.Mappings {
		a.Tools.VirtualDub.Mappings[i].Source = resolve(base, a.Tools.VirtualDub.Mappings[i].Source)
	}
}

func resolve(base, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(base, path))
}

func (a *App) validate() error {
	if strings.TrimSpace(a.Tools.FFprobe.Path) == "" {
		return fmt.Errorf("config key tools.ffprobe.path is required")
	}
	if strings.TrimSpace(a.Paths.Workflows) == "" {
		return fmt.Errorf("config key paths.workflows is required")
	}
	if strings.TrimSpace(a.Processing.OutputDir) == "" {
		return fmt.Errorf("config key processing.output_dir is required")
	}
	if strings.TrimSpace(a.Processing.WorkDir) == "" {
		return fmt.Errorf("config key processing.work_dir is required")
	}
	mode := strings.ToLower(strings.TrimSpace(a.Tools.VirtualDub.PathMode))
	if mode != "" && mode != "windows" && mode != "wine" {
		return fmt.Errorf("config key tools.virtualdub.path_mode must be windows or wine")
	}
	return nil
}
