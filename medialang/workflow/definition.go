package workflow

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

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
	Interlaced   *bool     `yaml:"interlaced,omitempty"`
	Width        int       `yaml:"width,omitempty"`
	Height       int       `yaml:"height,omitempty"`
	FPS          FloatList `yaml:"fps,omitempty"`
	FPSTolerance float64   `yaml:"fps_tolerance,omitempty"`
	PixelFormat  []string  `yaml:"pixel_format,omitempty"`
	ColorSpace   []string  `yaml:"color_space,omitempty"`
	Codec        []string  `yaml:"codec,omitempty"`
}

type FloatList []float64

func (f *FloatList) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.SequenceNode {
		var values []float64
		if err := node.Decode(&values); err != nil {
			return err
		}
		*f = FloatList(values)
		return nil
	}
	if node.Kind == yaml.ScalarNode {
		var value float64
		if err := node.Decode(&value); err != nil {
			return err
		}
		*f = FloatList{value}
		return nil
	}
	return fmt.Errorf("fps must be a number or a list of numbers")
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
