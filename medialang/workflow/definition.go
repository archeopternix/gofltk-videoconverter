package workflow

import "gopkg.in/yaml.v3"

type Definition struct {
	Version  int                    `yaml:"version"`
	ID       string                 `yaml:"id"`
	Name     string                 `yaml:"name"`
	Priority int                    `yaml:"priority"`
	Match    MatchDefinition        `yaml:"match"`
	Workflow []FilterNodeDefinition `yaml:"workflow"`
	Source   string                 `yaml:"-"`
}

type MatchDefinition struct {
	Interlaced   *bool    `yaml:"interlaced,omitempty"`
	Width        int      `yaml:"width,omitempty"`
	Height       int      `yaml:"height,omitempty"`
	FPS          float64  `yaml:"fps,omitempty"`
	FPSTolerance float64  `yaml:"fps_tolerance,omitempty"`
	PixelFormat  []string `yaml:"pixel_format,omitempty"`
	ColorSpace   []string `yaml:"color_space,omitempty"`
	Codec        []string `yaml:"codec,omitempty"`
}

type FilterNodeDefinition struct {
	ID      string    `yaml:"id"`
	Filter  string    `yaml:"filter"`
	Enabled *bool     `yaml:"enabled,omitempty"`
	Config  yaml.Node `yaml:"config,omitempty"`
}

func (n FilterNodeDefinition) IsEnabled() bool {
	return n.Enabled == nil || *n.Enabled
}
